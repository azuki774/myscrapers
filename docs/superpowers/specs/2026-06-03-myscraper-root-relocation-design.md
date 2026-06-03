# Myscraper Root Relocation Design

**Goal:** Promote the Go scraper to the primary implementation by moving it from `go/myscraper` to `myscraper/`, while leaving the existing Python scrapers isolated under `src/`.

## Repository Layout

- `myscraper/` becomes the active Go module and runtime home.
- `src/` remains the legacy Python area for `moneyforward` and `sbi`.
- Root-level tooling such as `flake.nix` continues to serve the whole repository.

## Migration Scope

- Move the current Go module, CLI entrypoint, internal packages, helper scripts, and Playwright cache location from `go/myscraper/` to `myscraper/`.
- Update Go module import paths from `github.com/azuki774/myscrapers/go/myscraper/...` to `github.com/azuki774/myscrapers/myscraper/...`.
- Update repository references in `.gitignore`, `flake.nix`, `README.md`, and the existing implementation plan so the documented path matches the real layout.

## Compatibility Expectations

- Existing Python scrapers remain untouched.
- The Go scaffold should still pass its current `go test ./...` suite when run from `myscraper/`.
- Nix shell setup should keep Playwright assets and Go module cache inside the repository checkout.
