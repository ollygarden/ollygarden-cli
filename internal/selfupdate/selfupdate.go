package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	latestReleaseURL   = "https://api.github.com/repos/ollygarden/ollygarden-cli/releases/latest"
	releaseDownloadURL = "https://github.com/ollygarden/ollygarden-cli/releases/download"
	releasePageURL     = "https://github.com/ollygarden/ollygarden-cli/releases/tag"
	latestReleasePage  = "https://github.com/ollygarden/ollygarden-cli/releases/latest"
	maxMetadataSize    = 1 << 20
	maxArchiveSize     = 100 << 20
	maxBinarySize      = 100 << 20
)

// Notice describes a newer stable release for passive user notification.
type Notice struct {
	Version string
	URL     string
}

// Options controls an explicit self-update operation.
type Options struct {
	CurrentVersion string
	Force          bool
	Progress       func(string)
}

// Result describes whether Run installed the latest stable release.
type Result struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Executable     string `json:"executable,omitempty"`
	Updated        bool   `json:"updated"`
	Forced         bool   `json:"forced"`
	CurrentIsNewer bool   `json:"current_is_newer,omitempty"`
}

type updater struct {
	options     Options
	httpClient  *http.Client
	latestURL   string
	downloadURL string
	githubToken string
	goos        string
	goarch      string
	executable  func() (string, os.FileMode, error)
	validate    func(context.Context, string, string) error
	replace     func(string, string) error
}

type releaseInfo struct {
	tag     string
	version string
}

// CheckLatest returns the latest stable release only when it is newer than
// currentVersion. Development builds and check failures produce no notice at
// the command layer; errors are returned so explicit callers can inspect them.
func CheckLatest(ctx context.Context, currentVersion string) (*Notice, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return checkLatestRelease(checkCtx, currentVersion, latestReleasePage, client)
}

// Run checks GitHub for the latest stable release and safely replaces the
// current executable when that release is newer.
func Run(ctx context.Context, options Options) (Result, error) {
	u := &updater{
		options:     options,
		httpClient:  &http.Client{Timeout: 2 * time.Minute},
		latestURL:   latestReleaseURL,
		downloadURL: releaseDownloadURL,
		githubToken: os.Getenv("GITHUB_TOKEN"),
		goos:        runtime.GOOS,
		goarch:      runtime.GOARCH,
		executable:  executableForUpdate,
		validate:    validateBinary,
		replace:     replaceExecutable,
	}
	return u.run(ctx)
}

func (u *updater) run(ctx context.Context) (Result, error) {
	current, err := canonicalVersion(u.options.CurrentVersion)
	if err != nil {
		return Result{}, fmt.Errorf("cannot update development build %q; reinstall with install.sh or `go install github.com/ollygarden/ollygarden-cli/cmd/ollygarden@latest`", u.options.CurrentVersion)
	}

	u.progress("Checking for updates...")
	release, err := u.latestRelease(ctx)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		CurrentVersion: displayVersion(current),
		LatestVersion:  displayVersion(release.version),
		Forced:         u.options.Force,
	}
	comparison := semver.Compare(release.version, current)
	if comparison < 0 {
		result.CurrentIsNewer = true
		return result, nil
	}
	if comparison == 0 && !u.options.Force {
		return result, nil
	}

	archiveName, binaryName, err := assetNames(displayVersion(release.version), u.goos, u.goarch)
	if err != nil {
		return Result{}, err
	}
	executable, mode, err := u.executable()
	if err != nil {
		return Result{}, err
	}
	result.Executable = executable

	tempDir, err := os.MkdirTemp(filepath.Dir(executable), ".ollygarden-update-")
	if err != nil {
		return Result{}, fmt.Errorf("creating update staging directory beside %s: %w", executable, err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	checksumURL := u.assetURL(release.tag, "checksums.txt")
	checksums, err := u.fetchBytes(ctx, checksumURL, maxMetadataSize, false)
	if err != nil {
		return Result{}, fmt.Errorf("downloading checksums: %w", err)
	}
	expected, err := checksumFor(checksums, archiveName)
	if err != nil {
		return Result{}, err
	}

	u.progress("Downloading " + archiveName + "...")
	archivePath := filepath.Join(tempDir, archiveName)
	actual, err := u.downloadFile(ctx, u.assetURL(release.tag, archiveName), archivePath, maxArchiveSize)
	if err != nil {
		return Result{}, fmt.Errorf("downloading %s: %w", archiveName, err)
	}
	if !strings.EqualFold(expected, actual) {
		return Result{}, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archiveName, expected, actual)
	}

	u.progress("Verifying downloaded release...")
	stagedPath := filepath.Join(tempDir, binaryName)
	if err := extractBinary(archivePath, stagedPath, binaryName, mode.Perm(), maxBinarySize); err != nil {
		return Result{}, fmt.Errorf("extracting %s: %w", archiveName, err)
	}
	if err := u.validate(ctx, stagedPath, release.version); err != nil {
		return Result{}, fmt.Errorf("validating downloaded executable: %w", err)
	}

	u.progress("Installing v" + displayVersion(release.version) + "...")
	if err := u.replace(stagedPath, executable); err != nil {
		return Result{}, fmt.Errorf("replacing %s: %w", executable, err)
	}
	result.Updated = true
	return result, nil
}

func (u *updater) progress(message string) {
	if u.options.Progress != nil {
		u.options.Progress(message)
	}
}

func checkLatestRelease(ctx context.Context, currentVersion, address string, client *http.Client) (*Notice, error) {
	current, err := canonicalVersion(currentVersion)
	if err != nil {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ollygarden/"+currentVersion)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("latest release page returned HTTP %d (%s)", resp.StatusCode, resp.Status)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("latest release redirect has no Location header")
	}
	redirect, err := req.URL.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("parsing latest release redirect: %w", err)
	}
	const tagMarker = "/releases/tag/"
	marker := strings.LastIndex(redirect.Path, tagMarker)
	if marker < 0 {
		return nil, fmt.Errorf("latest release redirect has unexpected path %q", redirect.Path)
	}
	tag, err := url.PathUnescape(strings.TrimPrefix(redirect.Path[marker:], tagMarker))
	if err != nil || tag == "" || strings.Contains(tag, "/") {
		return nil, fmt.Errorf("latest release redirect has invalid tag")
	}
	latest, err := canonicalVersion(tag)
	if err != nil {
		return nil, fmt.Errorf("latest release has invalid semantic version %q", tag)
	}
	if semver.Compare(latest, current) <= 0 {
		return nil, nil
	}
	return &Notice{
		Version: displayVersion(latest),
		URL:     strings.TrimRight(releasePageURL, "/") + "/" + url.PathEscape(tag),
	}, nil
}

func (u *updater) latestRelease(ctx context.Context) (releaseInfo, error) {
	body, err := u.fetchBytes(ctx, u.latestURL, maxMetadataSize, true)
	if err != nil {
		return releaseInfo{}, fmt.Errorf("checking latest GitHub release: %w", err)
	}
	var release struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return releaseInfo{}, fmt.Errorf("decoding latest GitHub release: %w", err)
	}
	if release.Draft || release.Prerelease {
		return releaseInfo{}, fmt.Errorf("GitHub returned a non-stable release as latest")
	}
	latest, err := canonicalVersion(release.TagName)
	if err != nil {
		return releaseInfo{}, fmt.Errorf("latest release has invalid semantic version %q", release.TagName)
	}
	return releaseInfo{tag: strings.TrimSpace(release.TagName), version: latest}, nil
}

func (u *updater) fetchBytes(ctx context.Context, address string, limit int64, authenticate bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ollygarden/"+u.options.CurrentVersion)
	if authenticate && u.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+u.githubToken)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := githubStatusError(resp); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

func (u *updater) downloadFile(ctx context.Context, address, path string, limit int64) (checksum string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "ollygarden/"+u.options.CurrentVersion)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := githubStatusError(resp); err != nil {
		return "", err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", err
	}
	if written > limit {
		return "", fmt.Errorf("archive exceeds %d bytes", limit)
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (u *updater) assetURL(tag, name string) string {
	return strings.TrimRight(u.downloadURL, "/") + "/" + url.PathEscape(tag) + "/" + url.PathEscape(name)
}

func githubStatusError(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return fmt.Errorf("GitHub API rate limit exceeded; set GITHUB_TOKEN and try again")
	}
	return fmt.Errorf("GitHub returned HTTP %d (%s)", resp.StatusCode, resp.Status)
}

func canonicalVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	core := strings.TrimPrefix(value, "v")
	if separator := strings.IndexAny(core, "+-"); separator >= 0 {
		core = core[:separator]
	}
	if strings.Count(core, ".") != 2 || !semver.IsValid(value) {
		return "", fmt.Errorf("invalid semantic version")
	}
	return value, nil
}

func displayVersion(value string) string {
	return strings.TrimPrefix(value, "v")
}

func assetNames(releaseVersion, goos, goarch string) (archive, binary string, err error) {
	if goarch != "amd64" && goarch != "arm64" {
		return "", "", fmt.Errorf("unsupported architecture %s", goarch)
	}
	switch goos {
	case "darwin", "linux":
		return fmt.Sprintf("ollygarden_%s_%s_%s.tar.gz", releaseVersion, goos, goarch), "ollygarden", nil
	case "windows":
		return fmt.Sprintf("ollygarden_%s_windows_%s.zip", releaseVersion, goarch), "ollygarden.exe", nil
	default:
		return "", "", fmt.Errorf("unsupported operating system %s", goos)
	}
}

func checksumFor(manifest []byte, archiveName string) (string, error) {
	for line := range strings.SplitSeq(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == archiveName {
			if len(fields[0]) != sha256.Size*2 {
				return "", fmt.Errorf("invalid checksum for %s", archiveName)
			}
			if _, err := hex.DecodeString(fields[0]); err != nil {
				return "", fmt.Errorf("invalid checksum for %s", archiveName)
			}
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksums.txt has no entry for %s", archiveName)
}

func executableForUpdate() (string, os.FileMode, error) {
	if invoked, err := exec.LookPath(os.Args[0]); err == nil {
		if info, statErr := os.Lstat(invoked); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", 0, fmt.Errorf("refusing to replace symlink %s; update ollygarden with the package manager or installer that created it", invoked)
		}
	}

	path, err := os.Executable()
	if err != nil {
		return "", 0, fmt.Errorf("locating current executable: %w", err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", 0, fmt.Errorf("resolving current executable %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, fmt.Errorf("inspecting current executable %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("current executable is not a regular file: %s", path)
	}
	return path, info.Mode(), nil
}

func validateBinary(ctx context.Context, path, expectedVersion string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(checkCtx, path, "version", "--json")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("running staged binary: %w", err)
	}
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return fmt.Errorf("reading staged binary version: %w", err)
	}
	actual, err := canonicalVersion(info.Version)
	if err != nil || semver.Compare(actual, expectedVersion) != 0 {
		return fmt.Errorf("staged binary reports version %q, expected %q", info.Version, displayVersion(expectedVersion))
	}
	return nil
}
