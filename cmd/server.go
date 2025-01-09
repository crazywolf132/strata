package cmd

import (
	"fmt"
	"strata/internal/logs"
	netsrv "strata/internal/net"

	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	srvCmd := &cobra.Command{
		Use:   "server",
		Short: "Run the optional Strata enterprise server (central ephemeral store).",
		Long: `This server can be deployed on-prem for enterprise.
It offers a secure store for shared stacks, token-based access, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			logs.Info("Starting optional enterprise server on port %d", port)
			fmt.Printf("Strata server running on port %d (Ctrl+C to stop)\n", port)
			return netsrv.StartServer(port)
		},
	}

	srvCmd.Flags().Int("port", 8080, "Port to run the Strata server on")
	return srvCmd
}
