package cli

import (
	"context"
	"dns-manager/internal/client"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newAddCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "add <server>",
		Short:   "Add DNS server",
		Example: "dnsctl add 8.8.8.8",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			server := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			apiClient := client.New(app.serverURL)

			servers, err := apiClient.AddServer(ctx, server)
			if err != nil {
				return fmt.Errorf("failed to add DNS server: %w", err)
			}

			fmt.Printf("DNS server added: %s\n", server)
			printServers(servers)

			return nil
		},
	}
}
