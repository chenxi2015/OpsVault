package nginx

import (
	"fmt"

	"OpsVault/internal/driver/binary"

	"github.com/spf13/cobra"
)

func (c *commandSet) newModuleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Manage Nginx dynamic modules",
	}
	cmd.AddCommand(c.newModuleListCommand(), c.newModuleAddCommand(), c.newModuleDelCommand())
	return cmd
}

func (c *commandSet) newModuleListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed dynamic modules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := c.driver()
			modules, err := d.ListModules()
			if err != nil {
				return err
			}
			if len(modules) == 0 {
				cmd.Println("No Nginx dynamic modules installed.")
				cmd.Println("Available preset aliases to add:")
				for preset, url := range binary.PresetModules {
					cmd.Printf("  - %-15s (%s)\n", preset, url)
				}
				return nil
			}

			cmd.Printf("%-20s %-40s %-10s %-10s\n", "NAME", "DYNAMIC LIBRARY (.so)", "ENABLED", "SIZE")
			cmd.Println("--------------------------------------------------------------------------------")
			for _, m := range modules {
				statusStr := "disabled"
				if m.Enabled {
					statusStr = "enabled"
				}
				sizeStr := fmt.Sprintf("%.2f MB", float64(m.Size)/(1024*1024))
				if m.Size < 1024*1024 {
					sizeStr = fmt.Sprintf("%.1f KB", float64(m.Size)/1024)
				}
				cmd.Printf("%-20s %-40s %-10s %-10s\n", m.Name, m.SoName, statusStr, sizeStr)
			}
			return nil
		},
	}
}

func (c *commandSet) newModuleAddCommand() *cobra.Command {
	var opts binary.AddModuleOptions
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Compile and install a dynamic Nginx module",
		Long: `Compile and install a dynamic Nginx module (.so) using incremental compilation (make modules).
Built-in preset module aliases:
  headers-more  (https://github.com/openresty/headers-more-nginx-module.git)
  echo          (https://github.com/openresty/echo-nginx-module.git)
  fancyindex    (https://github.com/aperezdc/ngx-fancyindex.git)
  brotli        (https://github.com/google/ngx_brotli.git)
  cache-purge   (https://github.com/FRiCKLE/ngx_cache_purge.git)

Examples:
  opsvault nginx module add --name headers-more
  opsvault nginx module add --name my-mod --url https://github.com/user/my-mod.git
  opsvault nginx module add --name my-mod --path /usr/local/src/my-mod`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().AddModule(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "module name (or preset alias: headers-more, echo, fancyindex, brotli, cache-purge)")
	cmd.Flags().StringVarP(&opts.URL, "url", "u", "", "git repository or tarball URL for module source code")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "local filesystem path to module source code")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func (c *commandSet) newModuleDelCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "del",
		Short: "Uninstall and disable a dynamic Nginx module",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().RemoveModule(name)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "dynamic module name to remove")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
