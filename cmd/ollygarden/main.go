package main

import "github.com/ollygarden/ollygarden-cli/cmd"

// Build metadata, populated at release time via ldflags:
//
//	-X main.version=... -X main.commit=... -X main.date=...
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetBuildInfo(version, commit, date)
	cmd.Execute()
}
