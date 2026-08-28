# AGENTS.md

## Purpose

This repository contains a Go-based scraper (`myscraper/`) for MoneyForward and SBI Securities.
Use this file as the default instruction set for AI agents working in this repo.

## Skills

- Check the repository-local skills under `.agents/skills/` and use them when they are relevant to the task.
- Prefer reusing existing skill guidance over inventing a new workflow for commits or reviews.

## Tooling

- Prefer running repository tools through Nix so commands use the expected versions and environment.
- The repo's `.envrc` is `use flake`, so `direnv allow` once per checkout (or per worktree) activates the same `nix develop` shell that `nix develop -c` would. After that, run repo tools directly (e.g. `go test -v ./...`, `gopls`, `playwright install chromium`) without prefixing every command with `nix develop -c`.
- `nix develop -c <command>` is still acceptable for one-off invocations from outside a direnv-activated shell, but in-session work should rely on the active shell.

## Go Changes

- If you modify Go code, run a self-test with a `go test` command before finishing the task.
- For the current Go scaffold, run tests from `myscraper` with `go test -v ./...`. The active direnv shell supplies the Go toolchain pinned in `flake.nix`.

## Pull Requests

- Open pull requests from a feature branch into `master` (or the agreed base branch).
- The PR description must follow `.github/PULL_REQUEST_TEMPLATE.md`. Use the sections defined there.
- Before opening a PR, load the `pr-template-filler` skill from `.agents/skills/` and follow it. The skill walks through the template and validates the description.
- Do not open a PR with empty sections, `TBD`, `TODO`, or `FIXME` placeholders. If a section does not apply, write a one-sentence reason instead of leaving it blank.
- Link the related issue, design doc, or implementation plan in the Note section. If none, write `None` and explain.
