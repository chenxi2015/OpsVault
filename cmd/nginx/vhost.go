package nginx

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

func (c *commandSet) newVHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vhost",
		Short: "Manage Nginx virtual hosts",
	}
	cmd.AddCommand(c.newVHostAddCommand(), c.newVHostDeleteCommand(), c.newVHostListCommand())
	return cmd
}

func (c *commandSet) newVHostAddCommand() *cobra.Command {
	var domain string
	var root string
	var proxy string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a virtual host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if root == "" {
				wwwRoot := c.config.GetString("nginx.www_root")
				if wwwRoot == "" {
					wwwRoot = "/data/wwwroot"
				}
				root = filepath.Join(wwwRoot, domain)
			}
			return c.driver().AddVHostWithOptions(domain, root, proxy)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "vhost domain")
	cmd.Flags().StringVar(&root, "root", "", "website root path (defaults to {nginx.www_root}/{domain} from config)")
	cmd.Flags().StringVarP(&proxy, "proxy", "p", "", "backend proxy address or port (e.g. 8080 or http://127.0.0.1:8080)")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func (c *commandSet) newVHostDeleteCommand() *cobra.Command {
	var domain string
	var deleteRoot bool
	cmd := &cobra.Command{
		Use:   "del",
		Short: "Delete a virtual host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().DeleteVHost(domain, deleteRoot)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "vhost domain")
	cmd.Flags().BoolVar(&deleteRoot, "delete-root", false, "delete site root")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func (c *commandSet) newVHostListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List virtual hosts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := c.driver().ListVHosts()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				cmd.Println("no vhosts found")
				return nil
			}
			for _, item := range items {
				cmd.Printf("%s\t%s\n", item["domain"], item["path"])
			}
			return nil
		},
	}
}
