package ansible

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGenerateInventoryFile(t *testing.T) {
	v := viper.New()
	v.Set("ansible.bin_path", "ansible")
	v.Set("ansible.temp_dir", "./test_tmp")

	groupsData := []map[string]interface{}{
		{
			"name": "db_group",
			"hosts": []map[string]interface{}{
				{
					"ip":               "192.168.1.100",
					"port":             2222,
					"user":             "dbuser",
					"ssh_private_key": "/path/to/key",
				},
			},
		},
		{
			"name": "web_group",
			"hosts": []map[string]interface{}{
				{
					"ip":           "192.168.1.200",
					"port":         22,
					"user":         "webuser",
					"ssh_password": "pwd",
				},
			},
		},
	}
	v.Set("ansible.inventory.groups", groupsData)

	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tempFile, err := GenerateInventoryFile(cfg)
	if err != nil {
		t.Fatalf("failed to generate inventory: %v", err)
	}
	defer os.RemoveAll("./test_tmp")

	contentBytes, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read generated inventory file: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "[db_group]") {
		t.Errorf("expected group [db_group] to exist in inventory")
	}
	if !strings.Contains(content, "192.168.1.100 ansible_port=2222 ansible_user=dbuser ansible_ssh_private_key_file=/path/to/key") {
		t.Errorf("expected host 192.168.1.100 details to match in inventory")
	}
	if !strings.Contains(content, "[web_group]") {
		t.Errorf("expected group [web_group] to exist in inventory")
	}
	if !strings.Contains(content, "192.168.1.200 ansible_port=22 ansible_user=webuser ansible_ssh_pass=pwd") {
		t.Errorf("expected host 192.168.1.200 details to match in inventory")
	}
}
