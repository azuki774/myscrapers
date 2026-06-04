package scrape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Browser interface {
	Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error)
}

type PageSnapshot struct {
	URL   string
	Title string
	HTML  string
}

type Request struct {
	URL        string
	OutputPath string
	Headless   bool
}

type Result struct {
	Title      string
	OutputPath string
}

type Service struct {
	Browser Browser
}

func (s Service) Run(ctx context.Context, req Request) (Result, error) {
	if s.Browser == nil {
		return Result{}, fmt.Errorf("browser is required")
	}

	snapshot, err := s.Browser.Fetch(ctx, req.URL, req.Headless)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(req.OutputPath, []byte(snapshot.HTML), 0o644); err != nil {
		return Result{}, err
	}

	return Result{
		Title:      snapshot.Title,
		OutputPath: req.OutputPath,
	}, nil
}
