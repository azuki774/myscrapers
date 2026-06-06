package moneyforward

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Cookie is the normalised cookie record that the rest of the package
// consumes. It mirrors the Selenium fields that the browser extension
// exports, with sameSite reduced to the three values Playwright accepts
// (Strict / Lax / None) and expires stored as a Unix-seconds float so the
// Playwright adapter can pass it through unchanged.
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	Expires  float64
	// SameSite is "Strict", "Lax", "None", or empty. Empty means the
	// source cookie had no sameSite field (or an unrecognised value);
	// the Playwright adapter skips passing sameSite to the browser in
	// that case, matching the Python reference behaviour.
	SameSite string
}

type rawCookie struct {
	Name           string  `json:"name"`
	Value          string  `json:"value"`
	Domain         string  `json:"domain"`
	Path           string  `json:"path"`
	Secure         bool    `json:"secure"`
	HTTPOnly       bool    `json:"httpOnly"`
	ExpirationDate float64 `json:"expirationDate"`
	SameSite       string  `json:"sameSite"`
}

// LoadCookies reads a browser-extension-style JSON cookie file and
// returns the subset of cookies that have name, value, and domain
// populated. The sameSite field is normalised to "Strict", "Lax",
// "None", or empty (empty when the source value is missing or
// unrecognised); cookies are kept regardless of sameSite to match the
// Python reference, and the Playwright adapter simply omits sameSite
// when the field is empty. An error is returned when the file does
// not exist, is invalid JSON, or contains no usable cookies.
func LoadCookies(path string) ([]Cookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cookie file: %w", err)
	}
	var raw []rawCookie
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse cookie file: %w", err)
	}
	out := make([]Cookie, 0, len(raw))
	for _, r := range raw {
		if r.Name == "" || r.Value == "" || r.Domain == "" {
			continue
		}
		c := Cookie{
			Name:     r.Name,
			Value:    r.Value,
			Domain:   r.Domain,
			Path:     r.Path,
			Secure:   r.Secure,
			HTTPOnly: r.HTTPOnly,
			Expires:  r.ExpirationDate,
			SameSite: normaliseSameSite(r.SameSite),
		}
		if c.Path == "" {
			c.Path = "/"
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable cookies in %s", path)
	}
	return out, nil
}

func normaliseSameSite(in string) string {
	switch strings.ToLower(in) {
	case "strict":
		return "Strict"
	case "lax":
		return "Lax"
	case "none", "no_restriction":
		return "None"
	default:
		return ""
	}
}
