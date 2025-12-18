package main

import "github.com/spf13/cobra"

func newResetCmd() *cobra.Command {
	var login string
	var ip string

	c := &cobra.Command{
		Use:   "reset",
		Short: "Reset bucket for login/ip",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return getClient(cmd).ResetBucket(cmd.Context(), login, ip)
		},
	}

	c.Flags().StringVar(&login, "login", "", "Login to reset")
	c.Flags().StringVar(&ip, "ip", "", "IP to reset")

	_ = c.MarkFlagRequired("login")
	_ = c.MarkFlagRequired("ip")

	return c
}
