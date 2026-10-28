package ansiblecmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"OpsVault/pkg/safety"
)

func TestResolveExecTarget_BothEmpty(t *testing.T) {
	_, err := resolveExecTarget("", "")
	if err == nil || !strings.Contains(err.Error(), "either --cmd/-c or --file/-f must be specified") {
		t.Fatalf("expected either error, got: %v", err)
	}
}

func TestResolveExecTarget_MutuallyExclusive(t *testing.T) {
	_, err := resolveExecTarget("uptime", "/path/to/script.sh")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestResolveExecTarget_Command(t *testing.T) {
	target, err := resolveExecTarget("uptime", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.isScript {
		t.Errorf("expected isScript=false, got true")
	}
	if target.module != "shell" {
		t.Errorf("expected module=shell, got %s", target.module)
	}
	if target.moduleArgs != "uptime" {
		t.Errorf("expected moduleArgs=uptime, got %s", target.moduleArgs)
	}
	if target.content != "uptime" {
		t.Errorf("expected content=uptime, got %s", target.content)
	}
	if !strings.Contains(target.auditDetail, "cmd=\"uptime\"") {
		t.Errorf("expected auditDetail with cmd, got: %s", target.auditDetail)
	}
}

func TestResolveExecTarget_FileNotFound(t *testing.T) {
	_, err := resolveExecTarget("", "/nonexistent/path/to/script.sh")
	if err == nil || !strings.Contains(err.Error(), "script file not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestResolveExecTarget_FileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := resolveExecTarget("", tmpDir)
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("expected is a directory error, got: %v", err)
	}
}

func TestResolveExecTarget_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.sh")
	if err := os.WriteFile(emptyFile, []byte("   \n\t  "), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := resolveExecTarget("", emptyFile)
	if err == nil || !strings.Contains(err.Error(), "script file is empty") {
		t.Fatalf("expected empty file error, got: %v", err)
	}
}

func TestResolveExecTarget_ValidScriptAndSafety(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "deploy.sh")
	scriptContent := "#!/bin/bash\necho 'hello world'\nrm -rf /data/opsvault/app\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}

	target, err := resolveExecTarget("", scriptPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !target.isScript {
		t.Errorf("expected isScript=true")
	}
	if target.module != "script" {
		t.Errorf("expected module=script, got %s", target.module)
	}
	absExpected, _ := filepath.Abs(scriptPath)
	if target.moduleArgs != absExpected {
		t.Errorf("expected moduleArgs=%s, got %s", absExpected, target.moduleArgs)
	}
	if target.content != scriptContent {
		t.Errorf("expected content match")
	}
	if !strings.Contains(target.auditDetail, "file=") {
		t.Errorf("expected auditDetail with file, got: %s", target.auditDetail)
	}

	// Verify safety evaluation identifies the dangerous pattern inside script
	assessment := safety.AssessShell(target.content)
	if assessment.Level != safety.RiskDangerous {
		t.Errorf("expected script with rm -rf to be assessed as dangerous, got: %s", assessment.Level)
	}
}

