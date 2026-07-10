package ansiblecmd

import (
	"fmt"
	"os"

	"OpsVault/internal/driver/ansible"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type commandSet struct {
	config *viper.Viper
}

// NewCommand creates the base ansible command and attaches its subcommands.
func NewCommand(cfg *viper.Viper) *cobra.Command {
	c := &commandSet{config: cfg}
	cmd := &cobra.Command{
		Use:   "ansible",
		Short: "Ansible batch orchestration and deployment commands",
		Long:  `Provide batch connection check, ad-hoc execution, system inspection, and service deployment across multiple nodes.`,
	}
	cmd.AddCommand(
		c.newPingCommand(),
		c.newExecCommand(),
		c.newDoctorCommand(),
		c.newDeployCommand(),
	)
	return cmd
}

func (c *commandSet) getExecutor() (*ansible.Executor, func(), error) {
	cfg, err := ansible.LoadConfig(c.config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load ansible config: %w", err)
	}
	inventoryFile, err := ansible.GenerateInventoryFile(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate inventory: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(inventoryFile)
	}
	return ansible.NewExecutor(cfg, inventoryFile), cleanup, nil
}
