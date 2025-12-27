package main

import "github.com/spf13/cobra"

type listSpec struct {
	name     string
	addFn    func(cmd *cobra.Command, cidr string) error
	removeFn func(cmd *cobra.Command, cidr string) error
}

func newListCmd(spec listSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.name,
		Short: "Manage " + spec.name + " CIDRs",
	}

	cmd.AddCommand(
		newListAddCmd(spec),
		newListRemoveCmd(spec),
	)

	return cmd
}

func newListAddCmd(spec listSpec) *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "add",
		Short: "Add CIDR to " + spec.name,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return spec.addFn(cmd, cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to add")
	_ = c.MarkFlagRequired("cidr")
	return c
}

func newListRemoveCmd(spec listSpec) *cobra.Command {
	var cidr string

	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove CIDR from " + spec.name,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return spec.removeFn(cmd, cidr)
		},
	}

	c.Flags().StringVar(&cidr, "cidr", "", "CIDR to remove")
	_ = c.MarkFlagRequired("cidr")
	return c
}
