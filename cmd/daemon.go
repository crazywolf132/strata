package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/daemon"
	"strata/internal/logs"
)

func newDaemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "Run the optional local Strata daemon (auto-sync, collaboration server).",
		Long: `Launches Strata's local daemon that periodically fetches updates, auto-syncs stacks,
and optionally hosts ephemeral data for stack sharing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs.Info("Starting Strata daemon in foreground...")
			fmt.Println("Starting Strata daemon (Ctrl+C to stop).")
			return daemon.Run()
		},
	}
}
