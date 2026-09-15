package figi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FIGILookup is the exchange-qualified identifier accepted by OpenFIGI.
// It is deliberately an internal input to the resolver and is not emitted
// in scraper result JSON.
type FIGILookup struct {
	Ticker   string
	ExchCode string
}

// Resolver resolves exchange-qualified tickers to composite FIGIs.
type Resolver interface {
	Resolve(ctx context.Context, lookups []FIGILookup) (map[FIGILookup]string, error)
}

type openFIGIRequest struct {
	IDType   string `json:"idType"`
	IDValue  string `json:"idValue"`
	ExchCode string `json:"exchCode"`
}

type openFIGIResult struct {
	Data    []openFIGIData `json:"data"`
	Error   string         `json:"error"`
	Warning string         `json:"warning"`
}

type openFIGIData struct {
	CompositeFIGI string `json:"compositeFIGI"`
	Ticker        string `json:"ticker"`
	ExchCode      string `json:"exchCode"`
}

// OpenFIGIClient is a small unauthenticated OpenFIGI HTTP client. Endpoint
// and HTTPClient are configurable so callers can test it without a network.
type OpenFIGIClient struct {
	HTTPClient *http.Client
	Endpoint   string
	sleep      func(context.Context, time.Duration) error
}

const openFIGIEndpoint = "https://api.openfigi.com/v3/mapping"

// NewOpenFIGIClient returns a client using the public OpenFIGI mapping API.
func NewOpenFIGIClient(httpClient *http.Client) *OpenFIGIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	} else if httpClient.Timeout == 0 {
		copy := *httpClient
		copy.Timeout = 15 * time.Second
		httpClient = &copy
	}
	return &OpenFIGIClient{
		HTTPClient: httpClient,
		Endpoint:   openFIGIEndpoint,
		sleep:      sleepWithContext,
	}
}

// NewOpenFIGIClientForEndpoint is useful for local integration tests and
// proxies. It still uses the same request and retry semantics.
func NewOpenFIGIClientForEndpoint(endpoint string, httpClient *http.Client) *OpenFIGIClient {
	c := NewOpenFIGIClient(httpClient)
	c.Endpoint = endpoint
	return c
}

// OpenFIGIResolver batches requests according to the public API limit and
// validates every response before returning any identifiers.
type OpenFIGIResolver struct {
	Client *OpenFIGIClient
}

// NewOpenFIGIResolver returns a resolver backed by OpenFIGI.
func NewOpenFIGIResolver(client *OpenFIGIClient) *OpenFIGIResolver {
	if client == nil {
		client = NewOpenFIGIClient(nil)
	}
	return &OpenFIGIResolver{Client: client}
}

// Resolve deduplicates inputs and sends at most ten mappings per request.
func (r *OpenFIGIResolver) Resolve(ctx context.Context, lookups []FIGILookup) (map[FIGILookup]string, error) {
	unique := make([]FIGILookup, 0, len(lookups))
	seen := make(map[FIGILookup]struct{}, len(lookups))
	for _, lookup := range lookups {
		lookup.Ticker = strings.TrimSpace(lookup.Ticker)
		lookup.ExchCode = strings.ToUpper(strings.TrimSpace(lookup.ExchCode))
		if lookup.Ticker == "" || lookup.ExchCode == "" {
			return nil, fmt.Errorf("invalid FIGI lookup: ticker and exchange are required")
		}
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		unique = append(unique, lookup)
	}
	resolved := make(map[FIGILookup]string, len(unique))
	for start := 0; start < len(unique); start += 10 {
		end := start + 10
		if end > len(unique) {
			end = len(unique)
		}
		batch, err := r.Client.mapping(ctx, unique[start:end])
		if err != nil {
			return nil, err
		}
		for i, result := range batch {
			lookup := unique[start+i]
			if len(result.Data) != 1 {
				return nil, fmt.Errorf("OpenFIGI returned %d results for %s/%s", len(result.Data), lookup.Ticker, lookup.ExchCode)
			}
			data := result.Data[0]
			if data.CompositeFIGI == "" || !strings.EqualFold(data.Ticker, lookup.Ticker) || !strings.EqualFold(data.ExchCode, lookup.ExchCode) {
				return nil, fmt.Errorf("OpenFIGI result mismatch for %s/%s", lookup.Ticker, lookup.ExchCode)
			}
			resolved[lookup] = data.CompositeFIGI
		}
	}
	return resolved, nil
}

func (c *OpenFIGIClient) mapping(ctx context.Context, lookups []FIGILookup) ([]openFIGIResult, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = NewOpenFIGIClient(nil).HTTPClient
	}
	if c.sleep == nil {
		c.sleep = sleepWithContext
	}
	requests := make([]openFIGIRequest, len(lookups))
	for i, lookup := range lookups {
		requests[i] = openFIGIRequest{IDType: "TICKER", IDValue: lookup.Ticker, ExchCode: lookup.ExchCode}
	}
	body, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("marshal OpenFIGI request: %w", err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create OpenFIGI request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("OpenFIGI request: %w", err)
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read OpenFIGI response: %w", readErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt == 3 {
				return nil, fmt.Errorf("OpenFIGI HTTP status %d after %d attempts", resp.StatusCode, attempt)
			}
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			if err := c.sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("OpenFIGI HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}
		var result []openFIGIResult
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, fmt.Errorf("decode OpenFIGI response: %w", err)
		}
		if len(result) != len(lookups) {
			return nil, fmt.Errorf("OpenFIGI response count %d, want %d", len(result), len(lookups))
		}
		for _, item := range result {
			if item.Warning != "" {
				return nil, fmt.Errorf("OpenFIGI mapping warning: %s", item.Warning)
			}
			if item.Error != "" {
				return nil, fmt.Errorf("OpenFIGI mapping error: %s", item.Error)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("OpenFIGI request failed")
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(strings.TrimSpace(retryAfter)); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
		return 0
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
