package nginx

import (
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
			return c.driver().ApplySSL(domain)
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
			return c.driver().RenewSSL(domain)
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
			return c.driver().DeleteSSL(domain)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "domain name")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}
