//go:build !windows

package selfupdate

import "os"

func replaceExecutable(stagedPath, executablePath string) error {
	return os.Rename(stagedPath, executablePath)
}
