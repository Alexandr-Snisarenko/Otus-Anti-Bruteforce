package main

import (
	"github.com/spf13/cobra"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Manage whitelist CIDRs",
	}

	cmd.AddCommand(
		newWhitelistAddCmd(),
		newWhitelistRemoveCmd(),
	)

	return cmd
}

func newWhitelistAddCmd() *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "add",
		Short: "Add CIDR to whitelist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getClient(cmd).AddToWhitelist(cmd.Context(), cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to add")
	_ = c.MarkFlagRequired("cidr")
	return c
}

func newWhitelistRemoveCmd() *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove CIDR from whitelist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getClient(cmd).RemoveFromWhitelist(cmd.Context(), cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to remove")
	_ = c.MarkFlagRequired("cidr")
	return c
}
