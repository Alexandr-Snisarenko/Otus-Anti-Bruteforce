package main

import (
	"github.com/spf13/cobra"
)

func newBlacklistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist",
		Short: "Manage blacklist CIDRs",
	}

	cmd.AddCommand(
		newBlacklistAddCmd(),
		newBlacklistRemoveCmd(),
	)

	return cmd
}

func newBlacklistAddCmd() *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "add",
		Short: "Add CIDR to blacklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getClient(cmd).AddToBlacklist(cmd.Context(), cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to add")
	_ = c.MarkFlagRequired("cidr")
	return c
}

func newBlacklistRemoveCmd() *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove CIDR from blacklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getClient(cmd).RemoveFromBlacklist(cmd.Context(), cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to remove")
	_ = c.MarkFlagRequired("cidr")
	return c
}
