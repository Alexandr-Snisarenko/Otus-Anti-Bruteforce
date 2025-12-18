package main

import (
	"context"
	"os"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/abfclient"
	"github.com/spf13/cobra"
)

type ctxKey string

const clientKey ctxKey = "abfclient"

var addr string

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "abfctl",
		Short: "Anti-bruteforce admin CLI",
		Example: `	abfctl --addr 127.0.0.1:50051 whitelist add --cidr 192.168.1.0/24 
	abfctl --addr 127.0.0.1:50051 check --login user --pass secret --ip 127.0.0.1
	abfctl --addr 127.0.0.1:50051 reset --login user --ip 127.0.0.1`,
		// Создаём клиента и кладём его в context перед выполнением любой команды
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c, err := abfclient.New(addr)
			if err != nil {
				return err
			}
			// кладём клиента в context команды
			ctx := context.WithValue(cmd.Context(), clientKey, c)
			cmd.SetContext(ctx)

			return nil
		},

		// Закрываем клиента после выполнения любой команды
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// достаём клиента и закрываем
			if c, ok := cmd.Context().Value(clientKey).(*abfclient.Client); ok && c != nil {
				return c.Close()
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(
		&addr,
		"addr",
		getenv("ABF_ADDR", "127.0.0.1:50051"),
		"gRPC address (or ABF_ADDR)",
	)

	root.AddCommand(newWhitelistCmd())
	root.AddCommand(newBlacklistCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newResetCmd())
	return root
}

func getClient(cmd *cobra.Command) *abfclient.Client {
	c, _ := cmd.Context().Value(clientKey).(*abfclient.Client)
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
