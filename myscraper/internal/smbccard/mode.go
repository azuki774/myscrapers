// Package smbccard owns SMBC Card specific automation. This stub is
// only present so internal/cli can reference the ModeWebMeisaiHTMLDump
// constant for option validation. The full credentials and workflow
// package lands in Task 2.
package smbccard

// ModeWebMeisaiHTMLDump identifies the CLI mode that logs in to Vpass
// and saves the WEB明細 page HTML snapshot. Task 2 owns the credential
// and automation logic; this constant is shared so the CLI can validate
// --mode values.
const ModeWebMeisaiHTMLDump = "smbc-card-webmeisai"
