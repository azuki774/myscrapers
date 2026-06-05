// This file owns the per-mode constants used by the CLI's --mode
// flag and the Service dispatch. Defining the constants here (in the
// scrape package) instead of in the cli package avoids a cli↔scrape
// import cycle: the cli package already imports scrape for the
// Request/Result types, so scrape cannot also import cli to share
// the constant. The cli package re-exports ModeFetchURL as
// cli.ModeFetchURL for callers that want the value through the cli
// package's namespace.
package scrape

// ModeFetchURL is the value of the --mode flag that fetches a single
// URL anonymously and dumps its HTML snapshot. It is also the empty
// value (Request.Mode == "") accepted by Service.Run for callers
// that do not set a mode.
const ModeFetchURL = "fetch-url"
