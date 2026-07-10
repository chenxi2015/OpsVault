package ansible

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratePlaybookFile(t *testing.T) {
	tempDir := "./test_playbook_tmp"
	defer os.RemoveAll(tempDir)

	vars := PlaybookVars{
		DataRoot:          "/data/opsvault",
		NetworkName:       "opsvault-net",
		CIDR:              "172.28.0.0/16",
		NamePrefix:        "opsvault",
		MySQLImage:        "mysql:8.0",
		MySQLPort:         3306,
		MySQLRootPassword: "rootpassword",
	}

	playbookPath, err := GeneratePlaybookFile(tempDir, "mysql", vars)
	if err != nil {
		t.Fatalf("failed to generate playbook: %v", err)
	}

	contentBytes, err := os.ReadFile(playbookPath)
	if err != nil {
		t.Fatalf("failed to read generated playbook: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "/data/opsvault/mysql") {
		t.Errorf("expected data root to be rendered in mysql playbook")
	}
	if !strings.Contains(content, "opsvault-mysql") {
		t.Errorf("expected mysql container name prefix to be rendered")
	}
	if !strings.Contains(content, "MYSQL_ROOT_PASSWORD=rootpassword") {
		t.Errorf("expected mysql root password to be rendered")
	}
}
