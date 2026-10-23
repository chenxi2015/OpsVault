package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNewInitModelIncludesJenkinsAndGitLab(t *testing.T) {
	m := newInitModel()
	if m == nil {
		t.Fatal("expected non-nil initModel")
	}

	hasJenkins := false
	hasGitLab := false
	for _, item := range m.items {
		if item.name == "jenkins" {
			hasJenkins = true
		}
		if item.name == "gitlab" {
			hasGitLab = true
		}
	}

	if !hasJenkins {
		t.Error("expected initModel to include 'jenkins'")
	}
	if !hasGitLab {
		t.Error("expected initModel to include 'gitlab'")
	}
}

func TestInitCommandConfigOnlyGeneratesAndRespectsForce(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := newInitCommand(viper.New(), nil)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-only", "--config-dir", tmpDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("opsvault init --config-only failed: %v", err)
	}

	defaultYaml := filepath.Join(tmpDir, "default.yaml")
	ansibleYaml := filepath.Join(tmpDir, "ansible.yaml")

	for _, f := range []string{defaultYaml, ansibleYaml} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("expected %s to be created, got error: %v", f, err)
		}
		if len(data) == 0 {
			t.Errorf("expected %s to be non-empty", f)
		}
	}

	// Modify default.yaml and verify re-running without --force does not overwrite it
	customContent := []byte("# custom user config\n")
	if err := os.WriteFile(defaultYaml, customContent, 0o644); err != nil {
		t.Fatalf("write custom default.yaml: %v", err)
	}

	out.Reset()
	cmd.SetArgs([]string{"--config-only", "--config-dir", tmpDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("second opsvault init --config-only failed: %v", err)
	}
	got, _ := os.ReadFile(defaultYaml)
	if string(got) != string(customContent) {
		t.Errorf("expected existing default.yaml to be preserved without --force")
	}
	if !strings.Contains(out.String(), "already exists, skipped") {
		t.Errorf("expected skipped message in output, got: %s", out.String())
	}

	// Re-run with --force and verify it overwrites with embedded template
	out.Reset()
	cmd.SetArgs([]string{"--config-only", "--config-dir", tmpDir, "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("opsvault init --config-only --force failed: %v", err)
	}
	gotAfterForce, _ := os.ReadFile(defaultYaml)
	if string(gotAfterForce) == string(customContent) {
		t.Errorf("expected --force to overwrite default.yaml with template")
	}
}

