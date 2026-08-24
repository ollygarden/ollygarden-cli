package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

func extractBinary(archivePath, targetPath, binaryName string, mode os.FileMode, limit int64) error {
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		return extractTarGz(archivePath, targetPath, binaryName, mode, limit)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, targetPath, binaryName, mode, limit)
	default:
		return fmt.Errorf("unsupported archive format")
	}
}

func extractTarGz(archivePath, targetPath, binaryName string, mode os.FileMode, limit int64) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if archiveEntryMatches(header.Name, binaryName) && header.Typeflag == tar.TypeReg {
			return writeBinary(targetPath, reader, mode, limit)
		}
	}
	return fmt.Errorf("archive does not contain %s", binaryName)
}

func extractZip(archivePath, targetPath, binaryName string, mode os.FileMode, limit int64) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	for _, file := range archive.File {
		if !archiveEntryMatches(file.Name, binaryName) || !file.FileInfo().Mode().IsRegular() {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			return err
		}
		err = writeBinary(targetPath, entry, mode, limit)
		closeErr := entry.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	return fmt.Errorf("archive does not contain %s", binaryName)
}

func archiveEntryMatches(entry, binaryName string) bool {
	return entry == binaryName || entry == "./"+binaryName
}

func writeBinary(targetPath string, source io.Reader, mode os.FileMode, limit int64) (err error) {
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := target.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(targetPath)
		}
	}()

	written, err := io.Copy(target, io.LimitReader(source, limit+1))
	if err != nil {
		return err
	}
	if written > limit {
		return fmt.Errorf("executable exceeds %d bytes", limit)
	}
	if err := target.Chmod(mode.Perm()); err != nil {
		return err
	}
	return target.Sync()
}
