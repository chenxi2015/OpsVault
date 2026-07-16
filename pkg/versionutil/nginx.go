// Package versionutil provides helpers to resolve the latest stable versions
// of third-party software (Nginx, OpenSSL, etc.) from their official sources.
package versionutil

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ResolveNginxVersion returns a concrete version string.
// If ver is empty or "latest", it fetches the latest stable release from
// nginx.org/en/download.html. Falls back to fallback on any error.
func ResolveNginxVersion(ver, fallback string) string {
	if ver == "" || strings.EqualFold(ver, "latest") {
		resolved, err := fetchLatestNginxVersion()
		if err != nil {
			log.Printf("[warn] failed to resolve latest Nginx version, falling back to %s: %v", fallback, err)
			return fallback
		}
		return resolved
	}
	return ver
}

// fetchLatestNginxVersion queries nginx.org/en/download.html and returns the
// first (latest) stable version string found, e.g. "1.27.4".
func fetchLatestNginxVersion() (string, error) {
	log.Println("[info] resolving latest stable Nginx version from nginx.org...")

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, "https://nginx.org/en/download.html", nil)
	req.Header.Set("User-Agent", "OpsVault/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("nginx.org request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nginx.org status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read nginx.org response: %w", err)
	}

	// Match hrefs like /download/nginx-1.27.4.tar.gz
	re := regexp.MustCompile(`nginx-(\d+\.\d+\.\d+)\.tar\.gz`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no nginx version found on nginx.org download page")
	}

	// The first match is the most recent stable release on the page
	latest := matches[0][1]
	log.Printf("[info] resolved Nginx latest stable version: %s", latest)
	return latest, nil
}
