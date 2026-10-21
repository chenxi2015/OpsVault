package nginxconf

import (
	"strings"
	"testing"
)

func TestRenderLogrotateDefault(t *testing.T) {
	cfg := Config{
		WWWLogsRoot: "/data/wwwlogs",
		RunUser:     "www",
		RunGroup:    "www",
	}

	content := RenderLogrotate(cfg)

	// Check glob matches all .log files in install logs and wwwlogs directories
	if !strings.Contains(content, "/usr/local/nginx/logs/*.log") || !strings.Contains(content, "/data/wwwlogs/*.log") {
		t.Fatalf("expected log glob pattern covering both install logs and wwwlogs, got:\n%s", content)
	}

	// Check default strategy is daily
	if !strings.Contains(content, "    daily") {
		t.Fatalf("expected daily directive, got:\n%s", content)
	}

	// Check retention count defaults to 30
	if !strings.Contains(content, "    rotate 30") {
		t.Fatalf("expected rotate 30, got:\n%s", content)
	}

	// Check dateext and dateformat
	if !strings.Contains(content, "    dateext") || !strings.Contains(content, "    dateformat -%Y%m%d") {
		t.Fatalf("expected dateext and daily dateformat, got:\n%s", content)
	}

	// Check compress and delaycompress
	if !strings.Contains(content, "    compress") || !strings.Contains(content, "    delaycompress") {
		t.Fatalf("expected compress and delaycompress, got:\n%s", content)
	}

	// Check create with user and group
	if !strings.Contains(content, "    create 0640 www www") {
		t.Fatalf("expected create 0640 www www, got:\n%s", content)
	}

	// Check USR1 signal reload
	if !strings.Contains(content, "kill -USR1") {
		t.Fatalf("expected kill -USR1 postrotate reload, got:\n%s", content)
	}
}

func TestRenderLogrotateSizeStrategy(t *testing.T) {
	cfg := Config{
		WWWLogsRoot:       "/data/wwwlogs/",
		RunUser:           "nginx",
		RunGroup:          "nginx",
		LogRotateStrategy: "size",
		LogRotateSize:     "50M",
		LogRotateRotate:   15,
	}

	content := RenderLogrotate(cfg)

	if !strings.Contains(content, "/data/wwwlogs/*.log") {
		t.Fatalf("trailing slash should be trimmed, got:\n%s", content)
	}

	if !strings.Contains(content, "    size 50M") {
		t.Fatalf("expected size 50M, got:\n%s", content)
	}

	if !strings.Contains(content, "    rotate 15") {
		t.Fatalf("expected rotate 15, got:\n%s", content)
	}

	// For size-based, dateformat should include timestamp suffix to avoid duplicate name conflicts
	if !strings.Contains(content, "    dateformat -%Y%m%d_%s") {
		t.Fatalf("expected timestamp dateformat for size strategy, got:\n%s", content)
	}

	if !strings.Contains(content, "    create 0640 nginx nginx") {
		t.Fatalf("expected create 0640 nginx nginx, got:\n%s", content)
	}
}

func TestRenderLogrotateBothStrategy(t *testing.T) {
	cfg := Config{
		WWWLogsRoot:       "/data/wwwlogs",
		LogRotateStrategy: "both",
		LogRotateSize:     "200M",
		LogRotateRotate:   60,
	}

	content := RenderLogrotate(cfg)

	if !strings.Contains(content, "    daily") || !strings.Contains(content, "    maxsize 200M") {
		t.Fatalf("expected both daily and maxsize 200M, got:\n%s", content)
	}

	if !strings.Contains(content, "    rotate 60") {
		t.Fatalf("expected rotate 60, got:\n%s", content)
	}
}

func TestRenderLogrotateDisableOptions(t *testing.T) {
	noCompress := false
	noDateExt := false
	cfg := Config{
		WWWLogsRoot:       "/data/wwwlogs",
		LogRotateCompress: &noCompress,
		LogRotateDateExt:  &noDateExt,
	}

	content := RenderLogrotate(cfg)

	if strings.Contains(content, "compress") {
		t.Fatalf("expected compression disabled, got:\n%s", content)
	}

	if strings.Contains(content, "dateext") {
		t.Fatalf("expected dateext disabled, got:\n%s", content)
	}
}
