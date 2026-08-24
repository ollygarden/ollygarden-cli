package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/selfupdate"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type versionCheckResult struct {
	notice *selfupdate.Notice
}

var (
	checkLatestVersion = selfupdate.CheckLatest
	stdoutIsTerminal   = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
	versionNoticeWait  = 250 * time.Millisecond
	versionCheck       <-chan versionCheckResult
)

func startVersionCheck(cmd *cobra.Command) {
	versionCheck = nil
	if !shouldCheckVersion(cmd) {
		return
	}

	result := make(chan versionCheckResult, 1)
	versionCheck = result
	check := checkLatestVersion
	currentVersion := version
	ctx := cmd.Context()
	go func() {
		notice, err := check(ctx, currentVersion)
		if err != nil {
			notice = nil
		}
		result <- versionCheckResult{notice: notice}
	}()
}

func showVersionNotice(cmd *cobra.Command) {
	result := versionCheck
	versionCheck = nil
	if result == nil {
		return
	}

	timer := time.NewTimer(versionNoticeWait)
	defer timer.Stop()
	select {
	case checked := <-result:
		if checked.notice == nil {
			return
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "\nUpdate Available\nNew version v%s is available. Run `ollygarden update`.\n\nChangelog: %s\n", checked.notice.Version, checked.notice.URL)
	case <-timer.C:
		// Passive checks never turn a successful command into a slow or failed one.
	}
}

func shouldCheckVersion(cmd *cobra.Command) bool {
	if cmd.Parent() == nil || version == "dev" || jsonMode || quiet || !stdoutIsTerminal() {
		return false
	}
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", "help":
			return false
		case "update", "version":
			if parent := current.Parent(); parent != nil && parent.Name() == "ollygarden" {
				return false
			}
		}
	}
	return true
}
