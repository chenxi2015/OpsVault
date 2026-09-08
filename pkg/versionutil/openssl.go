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

// DefaultGitHubProxies defines the default GitHub acceleration proxy prefixes for domestic environments.
var DefaultGitHubProxies = []string{
	"https://ghfast.top/",
	"https://mirror.ghproxy.com/",
	"https://ghproxy.net/",
	"https://hub.gitmirror.com/",
}

// GetOpenSSLDownloadURLs returns candidate download URLs for OpenSSL ordered by speed and stability for domestic/global servers.
// Optional customProxies will override or populate the proxy prefix list.
func GetOpenSSLDownloadURLs(version string, customProxies ...string) []string {
	version = strings.TrimPrefix(version, "openssl-")
	version = strings.TrimPrefix(version, "OpenSSL_")

	var githubURL, opensslCdnURL string
	if strings.HasPrefix(version, "1.") {
		tag := "OpenSSL_" + strings.ReplaceAll(version, ".", "_")
		githubURL = "https://github.com/openssl/openssl/releases/download/" + tag + "/openssl-" + version + ".tar.gz"
		opensslCdnURL = "https://www.openssl.org/source/old/1.1.1/openssl-" + version + ".tar.gz"
	} else {
		tag := "openssl-" + version
		githubURL = "https://github.com/openssl/openssl/releases/download/" + tag + "/openssl-" + version + ".tar.gz"
		opensslCdnURL = "https://www.openssl.org/source/openssl-" + version + ".tar.gz"
	}

	proxies := customProxies
	if len(proxies) == 0 {
		proxies = DefaultGitHubProxies
	}

	var candidates []string
	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if !strings.HasSuffix(proxy, "/") {
			proxy += "/"
		}
		candidates = append(candidates, proxy+githubURL)
	}

	candidates = append(candidates, opensslCdnURL)
	if !strings.HasPrefix(version, "1.") {
		candidates = append(candidates,
			"https://openssl-library.org/source/openssl-"+version+".tar.gz",
			"https://ftp.openssl.org/source/openssl-"+version+".tar.gz",
		)
	}
	candidates = append(candidates, githubURL)
	return candidates
}

// OpenSSLSourceURL returns the primary recommended download URL for OpenSSL.
func OpenSSLSourceURL(version string, customProxies ...string) string {
	urls := GetOpenSSLDownloadURLs(version, customProxies...)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
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
