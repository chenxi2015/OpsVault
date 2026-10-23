package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"OpsVault/pkg/logger"
	"OpsVault/pkg/versionutil"
)

const (
	// DefaultRepo is the official GitHub repository for OpsVault.
	DefaultRepo = "chenxi2015/opsvault"
)

// Options configures the self-update behavior.
type Options struct {
	Repo           string
	CurrentVersion string
	TargetTag      string
	CheckOnly      bool
	Force          bool
	Proxies        []string
	OS             string
	Arch           string
	ExecutablePath string
	HTTPClient     *http.Client
}

// Result holds the outcome of a version check or self-update operation.
type Result struct {
	CurrentVersion string
	TargetVersion  string
	ReleaseNotes   string
	Updated        bool
	AlreadyLatest  bool
	ExecutablePath string
}

// Updater handles checking, downloading, verifying, and atomically replacing the OpsVault binary.
type Updater struct {
	opts Options
}

// New creates a new Updater with sensible defaults filled in.
func New(opts Options) *Updater {
	if opts.Repo == "" {
		opts.Repo = DefaultRepo
	}
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}
	if len(opts.Proxies) == 0 {
		opts.Proxies = versionutil.DefaultGitHubProxies
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 30 * time.Minute}
	}
	return &Updater{opts: opts}
}

// Run executes the self-update template workflow:
// 1. Resolve target version -> 2. Compare version -> 3. Resolve binary path -> 4. Download & verify -> 5. Atomic replace.
func (u *Updater) Run() (*Result, error) {
	targetTag, notes, err := u.ResolveVersion(u.opts.TargetTag)
	if err != nil {
		return nil, fmt.Errorf("resolve target version: %w", err)
	}

	res := &Result{
		CurrentVersion: u.opts.CurrentVersion,
		TargetVersion:  targetTag,
		ReleaseNotes:   notes,
	}

	if !u.opts.Force && IsSameVersion(u.opts.CurrentVersion, targetTag) {
		res.AlreadyLatest = true
		return res, nil
	}

	if u.opts.CheckOnly {
		return res, nil
	}

	exePath, err := u.resolveExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	res.ExecutablePath = exePath

	if err := u.downloadAndReplace(targetTag, exePath); err != nil {
		return nil, err
	}

	res.Updated = true
	return res, nil
}

// ResolveVersion resolves "latest" or empty string to the newest GitHub release tag,
// or normalizes an explicit tag (e.g. "1.0.0" -> "v1.0.0").
func (u *Updater) ResolveVersion(tag string) (string, string, error) {
	tag = strings.TrimSpace(tag)
	if tag != "" && !strings.EqualFold(tag, "latest") {
		return NormalizeTag(tag), "", nil
	}

	resolvedTag, notes, err := u.fetchLatestViaAPI()
	if err == nil && resolvedTag != "" {
		return NormalizeTag(resolvedTag), notes, nil
	}
	logger.Debugf("GitHub API release check failed (%v), falling back to redirect probe...", err)

	resolvedTag, redirErr := u.fetchLatestViaRedirect()
	if redirErr == nil && resolvedTag != "" {
		return NormalizeTag(resolvedTag), "", nil
	}

	if err != nil {
		return "", "", fmt.Errorf("api error: %v; redirect fallback error: %w", err, redirErr)
	}
	return "", "", redirErr
}

// NormalizeTag ensures a semantic version tag starts with "v".
func NormalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" || strings.EqualFold(tag, "dev") || strings.EqualFold(tag, "latest") {
		return tag
	}
	if !strings.HasPrefix(tag, "v") {
		return "v" + tag
	}
	return tag
}

// IsSameVersion compares two version strings after normalizing "v" prefixes.
func IsSameVersion(current, target string) bool {
	c := strings.TrimSpace(current)
	t := strings.TrimSpace(target)
	if c == "" || t == "" || strings.EqualFold(c, "dev") {
		return false
	}
	return NormalizeTag(c) == NormalizeTag(t)
}

// AssetFileName returns the standalone release binary asset name for the configured OS and Arch.
func (u *Updater) AssetFileName() string {
	ext := ""
	if u.opts.OS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("opsvault-%s-%s%s", u.opts.OS, u.opts.Arch, ext)
}

func (u *Updater) fetchLatestViaAPI() (string, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.opts.Repo)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OpsVault-SelfUpdater/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	if payload.TagName == "" {
		return "", "", fmt.Errorf("empty tag_name in github api response")
	}
	return payload.TagName, payload.Body, nil
}

func (u *Updater) fetchLatestViaRedirect() (string, error) {
	releaseURL := fmt.Sprintf("https://github.com/%s/releases/latest", u.opts.Repo)
	candidates := u.buildCandidateURLs(releaseURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var lastErr error
	for _, url := range candidates {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "OpsVault-SelfUpdater/1.0")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		loc := resp.Header.Get("Location")
		resp.Body.Close()

		if tag := extractTagFromLocation(loc); tag != "" {
			return tag, nil
		}
		lastErr = fmt.Errorf("no release tag found in redirect location %q from %s (status %d)", loc, url, resp.StatusCode)
	}
	return "", lastErr
}

func extractTagFromLocation(location string) string {
	const marker = "/releases/tag/"
	idx := strings.Index(location, marker)
	if idx == -1 {
		return ""
	}
	tag := location[idx+len(marker):]
	tag = strings.Split(tag, "?")[0]
	tag = strings.Split(tag, "/")[0]
	return strings.TrimSpace(tag)
}

func (u *Updater) buildCandidateURLs(rawGitHubURL string) []string {
	var urls []string
	for _, proxy := range u.opts.Proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if !strings.HasSuffix(proxy, "/") {
			proxy += "/"
		}
		urls = append(urls, proxy+rawGitHubURL)
	}
	urls = append(urls, rawGitHubURL)
	return urls
}

func (u *Updater) resolveExecutablePath() (string, error) {
	exePath := u.opts.ExecutablePath
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return "", err
		}
	}
	if realPath, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = realPath
	}
	return filepath.Abs(exePath)
}

func (u *Updater) downloadAndReplace(tag, exePath string) error {
	assetName := u.AssetFileName()
	baseReleaseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", u.opts.Repo, tag)
	assetURLs := u.buildCandidateURLs(baseReleaseURL + "/" + assetName)
	checksumURLs := u.buildCandidateURLs(baseReleaseURL + "/checksums.txt")

	exeDir := filepath.Dir(exePath)
	tmpBinPath := filepath.Join(exeDir, fmt.Sprintf(".%s.new.%d", filepath.Base(exePath), os.Getpid()))
	tmpChecksumPath := filepath.Join(exeDir, fmt.Sprintf(".checksums.%d.txt", os.Getpid()))
	defer func() {
		_ = os.Remove(tmpBinPath)
		_ = os.Remove(tmpChecksumPath)
	}()

	logger.Infof("Downloading %s (%s)...", assetName, tag)
	if err := u.downloadWithFallback(tmpBinPath, assetURLs); err != nil {
		return fmt.Errorf("download binary %s: %w", assetName, err)
	}

	// Download and verify SHA256 checksum if checksums.txt is available
	if err := u.downloadWithFallback(tmpChecksumPath, checksumURLs); err == nil {
		checksumData, readErr := os.ReadFile(tmpChecksumPath)
		if readErr == nil {
			if err := VerifyFileChecksum(tmpBinPath, assetName, string(checksumData)); err != nil {
				return fmt.Errorf("checksum verification failed: %w", err)
			}
			logger.Infof("SHA256 checksum verified for %s", assetName)
		}
	} else {
		logger.Warnf("Could not download checksums.txt (%v), skipping SHA256 verification", err)
	}

	if err := os.Chmod(tmpBinPath, 0o755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	return AtomicReplaceBinary(exePath, tmpBinPath)
}

func (u *Updater) downloadWithFallback(targetPath string, candidateURLs []string) error {
	var lastErr error
	for _, url := range candidateURLs {
		if url == "" {
			continue
		}
		_ = os.Remove(targetPath)
		logger.Debugf("Attempting download from %s", url)

		if err := u.httpDownload(targetPath, url); err != nil {
			lastErr = err
			logger.Debugf("Download failed from %s: %v", url, err)
			continue
		}

		if info, err := os.Stat(targetPath); err == nil && info.Size() > 0 {
			return nil
		}
		lastErr = fmt.Errorf("downloaded file from %s is empty", url)
	}
	return fmt.Errorf("all download mirrors failed, last error: %w", lastErr)
}

func (u *Updater) httpDownload(targetPath, url string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "OpsVault-SelfUpdater/1.0")

	resp, err := u.opts.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return closeErr
	}
	return nil
}

// VerifyFileChecksum calculates the SHA256 of filePath and compares it against the expected entry in checksumsContent.
func VerifyFileChecksum(filePath, assetName, checksumsContent string) error {
	expectedHash := ParseChecksumForAsset(assetName, checksumsContent)
	if expectedHash == "" {
		return fmt.Errorf("no checksum entry found for %s in checksums.txt", assetName)
	}

	actualHash, err := ComputeFileSHA256(filePath)
	if err != nil {
		return err
	}

	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("sha256 mismatch for %s: expected %s, got %s", assetName, expectedHash, actualHash)
	}
	return nil
}

// ParseChecksumForAsset extracts the hex SHA256 digest for assetName from standard sha256sum output.
func ParseChecksumForAsset(assetName, checksumsContent string) string {
	lines := strings.Split(checksumsContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hash := fields[0]
		name := strings.TrimPrefix(fields[1], "*")
		name = filepath.Base(name)
		if name == assetName {
			return strings.ToLower(hash)
		}
	}
	return ""
}

// ComputeFileSHA256 returns the lowercase hex-encoded SHA256 hash of a file.
func ComputeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// AtomicReplaceBinary safely replaces currentExe with newExe using rename semantics
// to avoid ETXTBSY on running Linux binaries, rolling back automatically on error.
func AtomicReplaceBinary(currentExe, newExe string) error {
	backupExe := currentExe + ".old"
	_ = os.Remove(backupExe)

	hasCurrent := false
	if info, err := os.Stat(currentExe); err == nil {
		hasCurrent = true
		// Preserve original executable permission bits on the new binary
		_ = os.Chmod(newExe, info.Mode())
		if err := os.Rename(currentExe, backupExe); err != nil {
			return fmt.Errorf("backup current binary (%s -> %s): %w", currentExe, backupExe, err)
		}
	}

	if err := os.Rename(newExe, currentExe); err != nil {
		if hasCurrent {
			_ = os.Rename(backupExe, currentExe)
		}
		return fmt.Errorf("replace binary (%s -> %s): %w", newExe, currentExe, err)
	}

	// Best-effort cleanup of the backup binary (on Windows it may remain locked until process exits)
	_ = os.Remove(backupExe)
	return nil
}

// VerifyBinaryExecutable runs "<binary> --version" as a quick smoke test if supported on the current host.
func VerifyBinaryExecutable(binPath string) error {
	cmd := exec.Command(binPath, "--version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("binary smoke test failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

