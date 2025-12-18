package main

import "github.com/spf13/cobra"

func newResetCmd() *cobra.Command {
	var login string
	var ip string

	с := &cobra.Command{
		Use:   "reset",
		Short: "Reset bucket for login/ip",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getClient(cmd).ResetBucket(cmd.Context(), login, ip)
		},
	}

	с.Flags().StringVar(&login, "login", "", "Login to reset")
	с.Flags().StringVar(&ip, "ip", "", "IP to reset")

	_ = с.MarkFlagRequired("login")
	_ = с.MarkFlagRequired("ip")

	return с
}
