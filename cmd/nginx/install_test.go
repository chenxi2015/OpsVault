package nginx

import (
	"testing"

	"github.com/spf13/viper"
)

func TestInstallCommandFlags(t *testing.T) {
	cfg := viper.New()
	c := &commandSet{config: cfg}
	cmd := c.newInstallCommand()

	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatalf("expected --force flag to exist on nginx install command")
	}

	noStartFlag := cmd.Flags().Lookup("no-start")
	if noStartFlag == nil {
		t.Fatalf("expected --no-start flag to exist on nginx install command")
	}
	if noStartFlag.DefValue != "false" {
		t.Fatalf("expected --no-start default value to be 'false', got %s", noStartFlag.DefValue)
	}
}
