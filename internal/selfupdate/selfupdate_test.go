package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "release", input: "1.2.3", want: "v1.2.3"},
		{name: "v prefix", input: "v1.2.3", want: "v1.2.3"},
		{name: "prerelease", input: "1.2.3-rc.1", want: "v1.2.3-rc.1"},
		{name: "development", input: "dev", wantErr: true},
		{name: "incomplete", input: "1.2", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAssetNames(t *testing.T) {
	tests := []struct {
		goos, goarch string
		archive, bin string
		wantErr      bool
	}{
		{goos: "linux", goarch: "amd64", archive: "ollygarden_1.2.3_linux_amd64.tar.gz", bin: "ollygarden"},
		{goos: "darwin", goarch: "arm64", archive: "ollygarden_1.2.3_darwin_arm64.tar.gz", bin: "ollygarden"},
		{goos: "windows", goarch: "amd64", archive: "ollygarden_1.2.3_windows_amd64.zip", bin: "ollygarden.exe"},
		{goos: "freebsd", goarch: "amd64", wantErr: true},
		{goos: "linux", goarch: "riscv64", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			archive, bin, err := assetNames("1.2.3", tt.goos, tt.goarch)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.archive, archive)
			assert.Equal(t, tt.bin, bin)
		})
	}
}

func TestChecksumFor(t *testing.T) {
	hash := strings.Repeat("a", sha256.Size*2)
	got, err := checksumFor([]byte(hash+"  other.tar.gz\n"+hash+" *ollygarden.tar.gz\n"), "ollygarden.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, hash, got)

	_, err = checksumFor([]byte("not-a-hash  ollygarden.tar.gz\n"), "ollygarden.tar.gz")
	require.Error(t, err)
	_, err = checksumFor([]byte(hash+"  other.tar.gz\n"), "ollygarden.tar.gz")
	require.Error(t, err)
}

func TestExtractBinary(t *testing.T) {
	contents := []byte("downloaded executable")
	tests := []struct {
		name       string
		archive    string
		binaryName string
		build      func(*testing.T, string, string, []byte)
	}{
		{name: "tar gz", archive: "release.tar.gz", binaryName: "ollygarden", build: writeTarGz},
		{name: "zip", archive: "release.zip", binaryName: "ollygarden.exe", build: writeZip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, tt.archive)
			tt.build(t, archivePath, tt.binaryName, contents)
			targetPath := filepath.Join(dir, tt.binaryName)
			require.NoError(t, extractBinary(archivePath, targetPath, tt.binaryName, 0o751, 1024))
			got, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			assert.Equal(t, contents, got)
			info, err := os.Stat(targetPath)
			require.NoError(t, err)
			if runtime.GOOS != "windows" {
				assert.Equal(t, os.FileMode(0o751), info.Mode().Perm())
			}
		})
	}
}

func TestUpdaterNoopsWhenCurrent(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer server.Close()

	u := testUpdater(server.URL, Options{CurrentVersion: "1.2.3"})
	u.githubToken = "token"
	u.executable = func() (string, os.FileMode, error) {
		t.Fatal("current executable should not be inspected for a no-op update")
		return "", 0, nil
	}
	result, err := u.run(context.Background())
	require.NoError(t, err)
	assert.False(t, result.Updated)
	assert.Equal(t, "1.2.3", result.LatestVersion)
	assert.Equal(t, 1, requests)
}

func TestUpdaterDownloadsVerifiesAndReplaces(t *testing.T) {
	archiveName := "ollygarden_1.1.0_linux_amd64.tar.gz"
	archive := tarGzBytes(t, "ollygarden", []byte("new executable"))
	hash := sha256.Sum256(archive)

	var assetAuthenticated bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			_, _ = io.WriteString(w, `{"tag_name":"v1.1.0"}`)
		case "/download/v1.1.0/checksums.txt":
			assetAuthenticated = r.Header.Get("Authorization") != ""
			_, _ = fmt.Fprintf(w, "%x  %s\n", hash, archiveName)
		case "/download/v1.1.0/" + archiveName:
			assetAuthenticated = assetAuthenticated || r.Header.Get("Authorization") != ""
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	executable := filepath.Join(dir, "ollygarden")
	require.NoError(t, os.WriteFile(executable, []byte("old executable"), 0o751))
	var progress []string
	u := testUpdater(server.URL, Options{
		CurrentVersion: "1.0.0",
		Progress: func(message string) {
			progress = append(progress, message)
		},
	})
	u.githubToken = "token"
	u.executable = func() (string, os.FileMode, error) { return executable, 0o751, nil }
	u.validate = func(_ context.Context, path, expected string) error {
		assert.Equal(t, "v1.1.0", expected)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, []byte("new executable"), contents)
		return nil
	}
	u.replace = func(staged, target string) error {
		if err := os.Remove(target); err != nil {
			return err
		}
		return os.Rename(staged, target)
	}

	result, err := u.run(context.Background())
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Equal(t, executable, result.Executable)
	assert.False(t, assetAuthenticated, "GitHub token must only be sent to the API endpoint")
	contents, err := os.ReadFile(executable)
	require.NoError(t, err)
	assert.Equal(t, []byte("new executable"), contents)
	assert.Equal(t, []string{
		"Checking for updates...",
		"Downloading " + archiveName + "...",
		"Verifying downloaded release...",
		"Installing v1.1.0...",
	}, progress)
}

func TestUpdaterChecksumMismatchPreservesExecutable(t *testing.T) {
	archiveName := "ollygarden_1.1.0_linux_amd64.tar.gz"
	archive := tarGzBytes(t, "ollygarden", []byte("new executable"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			_, _ = io.WriteString(w, `{"tag_name":"v1.1.0"}`)
		case "/download/v1.1.0/checksums.txt":
			_, _ = fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", sha256.Size*2), archiveName)
		case "/download/v1.1.0/" + archiveName:
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	executable := filepath.Join(dir, "ollygarden")
	require.NoError(t, os.WriteFile(executable, []byte("old executable"), 0o755))
	u := testUpdater(server.URL, Options{CurrentVersion: "1.0.0"})
	u.executable = func() (string, os.FileMode, error) { return executable, 0o755, nil }
	u.validate = func(context.Context, string, string) error {
		t.Fatal("checksum mismatch must be detected before binary validation")
		return nil
	}
	u.replace = func(string, string) error {
		t.Fatal("checksum mismatch must be detected before replacement")
		return nil
	}

	_, err := u.run(context.Background())
	require.ErrorContains(t, err, "checksum mismatch")
	contents, readErr := os.ReadFile(executable)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("old executable"), contents)
}

func TestUpdaterRejectsDevelopmentBuildBeforeNetwork(t *testing.T) {
	u := testUpdater("http://127.0.0.1:1", Options{CurrentVersion: "dev"})
	_, err := u.run(context.Background())
	require.ErrorContains(t, err, "development build")
}

func TestCheckLatestReturnsNewerReleaseNotice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Location", "/ollygarden/ollygarden-cli/releases/tag/v1.1.0")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	notice, err := checkLatestRelease(context.Background(), "1.0.0", server.URL+"/releases/latest", noRedirectClient())
	require.NoError(t, err)
	require.NotNil(t, notice)
	assert.Equal(t, "1.1.0", notice.Version)
	assert.Equal(t, "https://github.com/ollygarden/ollygarden-cli/releases/tag/v1.1.0", notice.URL)
}

func TestCheckLatestReturnsNoNoticeForCurrentOrDevelopmentBuild(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Location", "/ollygarden/ollygarden-cli/releases/tag/v1.0.0")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	address := server.URL + "/releases/latest"

	notice, err := checkLatestRelease(context.Background(), "1.0.0", address, noRedirectClient())
	require.NoError(t, err)
	assert.Nil(t, notice)
	assert.Equal(t, 1, requests)

	notice, err = checkLatestRelease(context.Background(), "dev", address, noRedirectClient())
	require.NoError(t, err)
	assert.Nil(t, notice)
	assert.Equal(t, 1, requests, "development builds must not make a release request")
}

func TestCheckLatestRejectsInvalidRedirect(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		location string
	}{
		{name: "successful page", status: http.StatusOK},
		{name: "missing location", status: http.StatusFound},
		{name: "unexpected path", status: http.StatusFound, location: "/ollygarden/ollygarden-cli/releases"},
		{name: "invalid version", status: http.StatusFound, location: "/ollygarden/ollygarden-cli/releases/tag/latest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.location != "" {
					w.Header().Set("Location", tt.location)
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			_, err := checkLatestRelease(context.Background(), "1.0.0", server.URL+"/releases/latest", noRedirectClient())
			require.Error(t, err)
		})
	}
}

func TestUpdaterReleaseHTTPFailures(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			u := testUpdater(server.URL, Options{CurrentVersion: "1.0.0"})
			_, err := u.run(context.Background())
			require.ErrorContains(t, err, fmt.Sprintf("HTTP %d", status))
		})
	}
}

func TestGitHubRateLimitError(t *testing.T) {
	response := &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header)}
	response.Header.Set("X-RateLimit-Remaining", "0")
	require.ErrorContains(t, githubStatusError(response), "GITHUB_TOKEN")
}

func testUpdater(serverURL string, options Options) *updater {
	return &updater{
		options:     options,
		httpClient:  http.DefaultClient,
		latestURL:   serverURL + "/latest",
		downloadURL: serverURL + "/download",
		goos:        "linux",
		goarch:      "amd64",
		validate:    func(context.Context, string, string) error { return nil },
		replace:     func(string, string) error { return nil },
	}
}

func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func writeTarGz(t *testing.T, path, name string, contents []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, tarGzBytes(t, name, contents), 0o600))
}

func tarGzBytes(t *testing.T, name string, contents []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gz)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents)), Typeflag: tar.TypeReg}))
	_, err := tarWriter.Write(contents)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gz.Close())
	return buffer.Bytes()
}

func writeZip(t *testing.T, path, name string, contents []byte) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	entry, err := writer.Create(name)
	require.NoError(t, err)
	_, err = entry.Write(contents)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
}
