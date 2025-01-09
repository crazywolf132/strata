package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/config"
	"strata/internal/daemon"
	"strata/internal/git"
	"strata/internal/locks"
	"strata/internal/logs"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize Strata in the current repository",
		RunE: func(cmd *cobra.Command, args []string) error {

			locks.LockRepo()
			defer locks.UnlockRepo()

			if !git.IsGitRepo() {
				return fmt.Errorf("this directory is not a valid Git repository (missing .git)")
			}

			// Initialize global and local config
			if err := config.InitializeGlobalConfig(); err != nil {
				return err
			}
			if err := config.InitializeRepoConfig(); err != nil {
				return err
			}

			// Optionally register with the daemon, if runnign
			if daemon.IsDaemonRunning() {
				if err := daemon.RegisterRepo(); err != nil {
					return err
				}
				logs.Info("Registered repo with local daemon")
			}

			logs.Info("Strata initialized successfully. You're ready to stacking!")
			fmt.Println("Strata initialized successfully. You're ready to stacking!")

			return nil
		},
	}
}
