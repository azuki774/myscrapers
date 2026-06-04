---
name: pr-template-filler
description: Walk the agent through the repository's pull request template and validate the result before opening a PR. Use this skill immediately before opening a PR with `gh pr create` or when the user asks the agent to draft a PR description.
---

# Pull Request Template Filler

## Purpose

Make every pull request in this repository use the same template and contain the information reviewers need. The agent should produce a complete PR description, not a stub.

## When to load

- Before opening a PR with `gh pr create` (or the GitHub web UI)
- Before drafting a PR description for the user to review
- Whenever the user asks the agent to "open a PR", "draft a PR", or "prepare a PR"

## Source of truth

The template is `.github/PULL_REQUEST_TEMPLATE.md` at the repository root. Read that file at the start of this skill and treat it as the contract for what the PR description must contain.

## Required sections

The PR description must include, in this order, the following sections. Each must have real content, not placeholders.

- Summary
- Changes
- Test
- Note

## Walkthrough

1. Read `.github/PULL_REQUEST_TEMPLATE.md` to confirm the current section set.
2. Run `git status` and `git diff --stat` to understand the staged and unstaged changes that will go into the PR.
3. If a design doc exists at `docs/superpowers/specs/` or an implementation plan at `docs/superpowers/plans/`, read it for the Note section.
4. For each required section, draft concrete content based on the actual diff and any linked issue or plan.
5. If a section genuinely does not apply, say so explicitly (for example: "No documentation changes were needed."). Do not leave it blank.

## Validation

After drafting the PR description, check it against the following rules. If any are true, fix the draft before opening the PR.

- Any section is empty, contains only an `<!-- ... -->` comment, or contains placeholders such as `TBD`, `TODO`, `FIXME`, or `N/A` without an explanation.
- The Summary is missing, is a single word, or is just a restatement of the commit message subject.
- The Changes section does not list the user-visible changes (a bullet list is preferred).
- The Test section does not list at least one concrete verification step (a command run, a scenario tested, or output from `go test`, `pytest`, or similar).
- The Note section does not state the rationale for the change, link an issue, design doc, or implementation plan, or explain why none exist. If the PR is exploratory, write `None` and add one sentence explaining why.

## Commands

Run these from the repository root before opening the PR:

```sh
git status
git diff --stat
git log --oneline -10
gh pr create --draft --fill --body-file <prepared-body>.md
```

## Subagent instructions

The following prompt can be used to dispatch a subagent to fill the PR template.

```text
You are a pull request template filler for this repository.
Read .github/PULL_REQUEST_TEMPLATE.md and treat it as the contract.
Draft a PR description for the current branch.
For each required section (Summary, Changes, Test, Note), write real content based on the diff and any linked issue, design doc, or implementation plan.
Do not leave placeholders such as TBD, TODO, FIXME, or empty HTML comments. If a section does not apply, say so in one sentence.
After drafting, validate the description against the rules in the pr-template-filler skill and fix any issues before returning.
Return the full PR description as markdown, ready to paste into `gh pr create --body-file`.
```

## Out of scope

- CI-side enforcement (linting the PR body in GitHub Actions)
- Multiple templates per change type
- Backporting older PRs to the new template
- Auto-generating the PR title (use the `mistship-conventional-commit-writer` skill for commit message style)
