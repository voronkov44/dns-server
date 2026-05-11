package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultServerURL = "http://localhost:8080"

type App struct {
	serverURL string
}

func NewRootCommand() *cobra.Command {
	app := &App{}

	rootCmd := &cobra.Command{
		Use:   "dnsctl",
		Short: "CLI client for dns-manager server",
		Long: `dnsctl is a CLI client for managing DNS servers through dns-manager REST API.

The server application modifies resolv.conf on the remote machine.
The CLI does not edit resolv.conf directly; it only sends requests to the server.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(
		&app.serverURL,
		"server",
		getEnv("DNS_MANAGER_SERVER_URL", defaultServerURL),
		"DNS manager server URL",
	)

	rootCmd.AddCommand(newListCommand(app))
	rootCmd.AddCommand(newAddCommand(app))
	rootCmd.AddCommand(newDeleteCommand(app))

	return rootCmd
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
