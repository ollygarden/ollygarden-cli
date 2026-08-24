//go:build windows

package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func replaceExecutable(stagedPath, executablePath string) error {
	backupPath := filepath.Join(filepath.Dir(executablePath), "."+filepath.Base(executablePath)+".old")
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing previous update backup: %w", err)
	}
	if err := os.Rename(executablePath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(stagedPath, executablePath); err != nil {
		if rollbackErr := os.Rename(backupPath, executablePath); rollbackErr != nil {
			return fmt.Errorf("install failed: %w (rollback also failed: %v)", err, rollbackErr)
		}
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		hideFile(backupPath)
	}
	return nil
}

func hideFile(path string) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return
	}
	_ = windows.SetFileAttributes(pointer, attributes|windows.FILE_ATTRIBUTE_HIDDEN)
}
