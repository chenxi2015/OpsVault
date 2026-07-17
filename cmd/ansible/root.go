package ansiblecmd

import (
	"fmt"
	"os"
	"path/filepath"

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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Trigger parent PersistentPreRunE first to perform base setup (e.g. initConfig, logging)
			if parent := cmd.Parent(); parent != nil && parent.PersistentPreRunE != nil {
				if err := parent.PersistentPreRunE(parent, args); err != nil {
					return err
				}
			}

			env, err := cmd.Flags().GetString("env")
			if err != nil {
				return err
			}

			configName := "ansible"
			if env != "" {
				configName = "ansible." + env
			}

			ansibleCfg := viper.New()
			ansibleCfg.SetConfigType("yaml")
			ansibleCfg.SetConfigName(configName)
			ansibleCfg.AddConfigPath("./configs")
			ansibleCfg.AddConfigPath(".")
			if home, err := os.UserHomeDir(); err == nil {
				ansibleCfg.AddConfigPath(filepath.Join(home, ".opsvault"))
			}

			if err := ansibleCfg.ReadInConfig(); err != nil {
				// Require the file if environment was explicitly specified.
				// If not specified (defaulting to ansible.yaml), fallback silently.
				if env != "" {
					return fmt.Errorf("failed to read %s.yaml config: %w", configName, err)
				}
			} else {
				if err := cfg.MergeConfigMap(ansibleCfg.AllSettings()); err != nil {
					return fmt.Errorf("failed to merge %s.yaml config: %w", configName, err)
				}
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringP("env", "e", "", "ansible environment configuration (e.g. dev, test, prod)")

	cmd.AddCommand(
		c.newListCommand(),
		c.newPingCommand(),
		c.newExecCommand(),
		c.newDoctorCommand(),
		c.newDeployCommand(),
		c.newPushCommand(),
		c.newUninstallCommand(),
		c.newReloadCommand(),
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
