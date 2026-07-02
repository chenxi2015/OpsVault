package oneinstack

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"OpsVault/pkg/fileutil"

	"github.com/spf13/viper"
)

type Downloader struct {
	config *viper.Viper
	client *http.Client
}

func NewDownloader(cfg *viper.Viper) *Downloader {
	return &Downloader{
		config: cfg,
		client: http.DefaultClient,
	}
}

func (d *Downloader) DownloadAutoScript(targetDir string) (string, error) {
	if err := fileutil.EnsureDir(targetDir, 0o755); err != nil {
		return "", err
	}
	scriptURL := d.config.GetString("oneinstack.auto_script_url")
	if scriptURL == "" {
		return "", fmt.Errorf("oneinstack.auto_script_url is required")
	}
	req, err := http.NewRequest(http.MethodGet, scriptURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download oneinstack script: unexpected status %s", resp.Status)
	}

	path := filepath.Join(targetDir, scriptFileNameFromURL(scriptURL))
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func scriptFileNameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "oneinstack-auto.sh"
	}
	base := filepath.Base(strings.TrimSuffix(parsed.Path, "/"))
	if base == "." || base == "/" || base == "" || !strings.Contains(base, ".") {
		return "oneinstack-auto.sh"
	}
	return base
}
