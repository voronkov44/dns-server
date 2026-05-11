package cli

import (
	"context"
	"dns-manager/internal/client"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newListCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all DNS servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			apiClient := client.New(app.serverURL)

			servers, err := apiClient.ListServers(ctx)
			if err != nil {
				return fmt.Errorf("failed to list DNS servers: %w", err)
			}

			printServers(servers)

			return nil
		},
	}
}

func printServers(servers []string) {
	if len(servers) == 0 {
		fmt.Println("No DNS servers configured")
		return
	}

	fmt.Println("DNS servers:")
	for _, server := range servers {
		fmt.Printf("- %s\n", server)
	}
}
