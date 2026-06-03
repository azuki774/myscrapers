# Myscraper Root Relocation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the Go scraper scaffold from `go/myscraper` to `myscraper/` and update the repository so the new path is the canonical location.

**Architecture:** Keep the current Go module structure intact, but promote it to a root-level `myscraper/` directory to distinguish it from the legacy Python `src/` tree. Update code imports, shell configuration, ignore rules, and written docs so all paths consistently point at the new module home.

**Tech Stack:** Go 1.24, Nix flakes, Playwright-Go scaffold, Markdown docs

---

### Task 1: Capture the Approved Layout Change

**Files:**
- Create: `docs/superpowers/specs/2026-06-03-myscraper-root-relocation-design.md`
- Create: `docs/superpowers/plans/2026-06-03-myscraper-root-relocation.md`

- [ ] **Step 1: Write the approved design summary**

Write a short spec that records:
- `myscraper/` is the active Go module location
- `src/` remains the legacy Python area
- module/import/tooling references must move from `go/myscraper` to `myscraper`

- [ ] **Step 2: Save the execution plan**

Record this relocation plan under `docs/superpowers/plans/2026-06-03-myscraper-root-relocation.md` so the implementation scope and verification steps are explicit.

### Task 2: Move the Go Module and Update References

**Files:**
- Move: `go/myscraper/go.mod` -> `myscraper/go.mod`
- Move: `go/myscraper/cmd/myscraper/main.go` -> `myscraper/cmd/myscraper/main.go`
- Move: `go/myscraper/internal/cli/run.go` -> `myscraper/internal/cli/run.go`
- Move: `go/myscraper/internal/cli/run_test.go` -> `myscraper/internal/cli/run_test.go`
- Move: `go/myscraper/internal/browser/.gitkeep` -> `myscraper/internal/browser/.gitkeep`
- Move: `go/myscraper/scripts/dev-shell-smoke.sh` -> `myscraper/scripts/dev-shell-smoke.sh`
- Modify: `.gitignore`
- Modify: `flake.nix`
- Modify: `README.md`
- Modify: `docs/superpowers/plans/2026-06-03-myscraper-go-playwright-bootstrap.md`

- [ ] **Step 1: Move the directory tree into its new root-level home**

Relocate the existing tracked files from `go/myscraper/` to `myscraper/` without changing their internal structure.

- [ ] **Step 2: Update the Go module path and imports**

Change:
- `module github.com/azuki774/myscrapers/go/myscraper`

To:
- `module github.com/azuki774/myscrapers/myscraper`

Update all in-repo Go imports to use the new module path.

- [ ] **Step 3: Update repo tooling and docs**

Replace `go/myscraper` references with `myscraper` in:
- ignore rules for `.playwright`, `tmp`, and built binaries
- `flake.nix` Playwright cache path
- the root README guidance
- the prior bootstrap plan so future work follows the new layout

### Task 3: Verify the Relocated Module

**Files:**
- Test: `myscraper/internal/cli/run_test.go`

- [ ] **Step 1: Run the Go test suite from the new module root**

Run: `nix develop -c go test ./...`
Working directory: `myscraper/`
Expected: exit 0, current Go tests pass under the relocated module path.

- [ ] **Step 2: Check for stale path references**

Run: `rg -n "go/myscraper" .`
Expected: only intentional historical mentions remain, or no matches after doc updates.

### Task 4: Publish the Change

**Files:**
- Stage only relocation-related files

- [ ] **Step 1: Review the scoped diff**

Confirm `.vscode/` and any unrelated files are excluded from staging.

- [ ] **Step 2: Commit the relocation**

Use a focused commit message describing the path promotion, for example:

```bash
git commit -m "refactor: move myscraper module to repo root"
```

- [ ] **Step 3: Push and open a draft PR**

Push the current branch and create a draft PR summarizing:
- the new `myscraper/` layout
- updated imports and tooling references
- verification performed
