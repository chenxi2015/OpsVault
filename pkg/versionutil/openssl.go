package versionutil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ResolveOpenSSLVersion returns a concrete version string for OpenSSL.
// If ver is empty or "latest", it fetches the latest stable 3.x release from GitHub Releases API.
// Falls back to fallback (e.g. "3.0.15") on any error.
func ResolveOpenSSLVersion(ver, fallback string) string {
	if ver == "" || strings.EqualFold(ver, "latest") {
		resolved, err := fetchLatestOpenSSLVersion()
		if err != nil {
			log.Printf("[warn] failed to resolve latest OpenSSL version, falling back to %s: %v", fallback, err)
			return fallback
		}
		return resolved
	}
	return ver
}

// OpenSSLSourceURL returns the GitHub Releases download URL for the given OpenSSL version.
// Handles both 1.x tags (OpenSSL_1_1_1w) and 3.x tags (openssl-3.0.15).
func OpenSSLSourceURL(version string) string {
	version = strings.TrimPrefix(version, "openssl-")
	version = strings.TrimPrefix(version, "OpenSSL_")
	if strings.HasPrefix(version, "1.") {
		tag := "OpenSSL_" + strings.ReplaceAll(version, ".", "_")
		return "https://github.com/openssl/openssl/releases/download/" + tag + "/openssl-" + version + ".tar.gz"
	}
	tag := "openssl-" + version
	return "https://github.com/openssl/openssl/releases/download/" + tag + "/openssl-" + version + ".tar.gz"
}

func fetchLatestOpenSSLVersion() (string, error) {
	log.Println("[info] resolving latest stable OpenSSL version from GitHub...")

	type release struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/openssl/openssl/releases?per_page=30", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OpsVault/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api status: %s", resp.Status)
	}

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("decode github response: %w", err)
	}

	re := regexp.MustCompile(`^openssl-(3\.\d+\.\d+)$`)
	for _, r := range releases {
		if r.Prerelease || r.Draft {
			continue
		}
		if m := re.FindStringSubmatch(r.TagName); len(m) == 2 {
			log.Printf("[info] resolved OpenSSL latest stable version: %s", m[1])
			return m[1], nil
		}
	}
	return "", fmt.Errorf("no stable OpenSSL 3.x release found in GitHub API response")
}
