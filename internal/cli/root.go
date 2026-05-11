package cli

import (
	"dns-manager/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type App struct {
	serverURL string
}

func NewRootCommand() *cobra.Command {
	cfg := config.LoadConfig()

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
		cfg.ServerURL,
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
