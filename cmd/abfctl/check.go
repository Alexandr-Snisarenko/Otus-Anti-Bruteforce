package main

import (
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var login, pass, ip string
	c := &cobra.Command{
		Use:   "check",
		Short: "Check if request is allowed",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ok, err := getClient(cmd).Check(cmd.Context(), login, pass, ip)
			if err != nil {
				return err
			}
			if ok {
				cmd.Println("Request is allowed")
			} else {
				cmd.Println("Request is not allowed")
			}
			return nil
		},
	}

	c.Flags().StringVar(&login, "login", "", "Login to check")
	_ = c.MarkFlagRequired("login")

	c.Flags().StringVar(&pass, "pass", "", "Password to check")
	_ = c.MarkFlagRequired("pass")

	c.Flags().StringVar(&ip, "ip", "", "IP address to check")
	_ = c.MarkFlagRequired("ip")

	return c
}
