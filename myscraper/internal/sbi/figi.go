package sbi

import (
	"net/http"

	"github.com/azuki774/myscrapers/myscraper/internal/figi"
)

type FIGILookup = figi.FIGILookup
type FIGIResolver = figi.Resolver
type OpenFIGIClient = figi.OpenFIGIClient
type OpenFIGIResolver = figi.OpenFIGIResolver

func NewOpenFIGIClient(h *http.Client) *OpenFIGIClient { return figi.NewOpenFIGIClient(h) }
func NewOpenFIGIClientForEndpoint(e string, h *http.Client) *OpenFIGIClient {
	return figi.NewOpenFIGIClientForEndpoint(e, h)
}
func NewOpenFIGIResolver(c *OpenFIGIClient) *OpenFIGIResolver { return figi.NewOpenFIGIResolver(c) }
