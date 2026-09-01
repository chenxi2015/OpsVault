package qdrant

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Qdrant vector database",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if apiKey == "" {
				apiKey = c.config.GetString("qdrant.api_key")
			}
			drv, err := c.driver(apiKey)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				return err
			}
			credutil.PrintCredentials("Qdrant Vector DB", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Qdrant authentication API key")
	return cmd
}
