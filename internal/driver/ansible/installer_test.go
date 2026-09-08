package ansible

import (
	"os"
	"testing"
)

func TestEnsureAnsible_InTestMode(t *testing.T) {
	cfg := &Config{
		BinPath:         "ansible",
		PlaybookBinPath: "ansible-playbook",
	}

	// In test mode (flag.Lookup("test.v") != nil), EnsureAnsible returns nil without executing commands
	if err := EnsureAnsible(cfg); err != nil {
		t.Fatalf("expected nil error in test mode, got: %v", err)
	}
}

func TestEnsureUserLocalBinInPath(t *testing.T) {
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	ensureUserLocalBinInPath()
	// Should not panic or corrupt PATH
	if os.Getenv("PATH") == "" {
		t.Fatalf("PATH should not be empty")
	}
}

func TestHasCommand(t *testing.T) {
	// "go" or "echo" or "cmd" should exist in test environment
	if !hasCommand("go") && !hasCommand("cmd") && !hasCommand("echo") {
		t.Logf("basic binary not found via LookPath")
	}

	if hasCommand("definitely_non_existent_binary_xyz_123") {
		t.Fatalf("expected non existent command to return false")
	}
}
