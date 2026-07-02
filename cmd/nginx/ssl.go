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
			manager := sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}
			if err := manager.Apply(domain, root); err != nil {
				return err
			}
			return c.driver().EnableSSL(domain)
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
			manager := sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}
			if err := manager.Renew(domain); err != nil {
				return err
			}
			return c.driver().Reload()
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
			manager := sslutil.Manager{SSLRoot: c.config.GetString("oneinstack.ssl_root")}
			if err := manager.Delete(domain); err != nil {
				return err
			}
			return c.driver().DisableSSL(domain)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "domain name")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}
