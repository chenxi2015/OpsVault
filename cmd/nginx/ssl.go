package nginx

import (
	"path/filepath"

	"OpsVault/pkg/sslutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newSSLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
	}
	cmd.AddCommand(c.newSSLApplyCommand(), c.newSSLRenewCommand(), c.newSSLDeleteCommand())
	return cmd
}

func (c *commandSet) newSSLApplyCommand() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a Let's Encrypt certificate",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := filepath.Join(c.config.GetString("oneinstack.www_root"), domain)
			return sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}.Apply(domain, root)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "domain name")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func (c *commandSet) newSSLRenewCommand() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "renew",
		Short: "Renew SSL certificates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}.Renew(domain)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "domain name")
	return cmd
}

func (c *commandSet) newSSLDeleteCommand() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete SSL certificates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}.Delete(domain)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "domain name")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}
