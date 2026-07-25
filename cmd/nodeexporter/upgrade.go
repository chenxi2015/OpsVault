package nodeexporter

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUpgradeCommand() *cobra.Command {
	var tag string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Node Exporter container image",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			targetImage := tag
			if targetImage != "" && !hasTagFormat(targetImage) {
				targetImage = "prom/node-exporter:" + tag
			}
			return drv.Upgrade(targetImage)
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "Target Node Exporter image tag (e.g. latest, v1.8.1)")
	return cmd
}

func hasTagFormat(img string) bool {
	for i := 0; i < len(img); i++ {
		if img[i] == ':' {
			return true
		}
	}
	return false
}
