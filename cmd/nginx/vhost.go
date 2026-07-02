package nginx

import "github.com/spf13/cobra"

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
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a virtual host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().AddVHost(domain, root)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "vhost domain")
	cmd.Flags().StringVar(&root, "root", "", "website root path")
	_ = cmd.MarkFlagRequired("domain")
	_ = cmd.MarkFlagRequired("root")
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
