package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/git"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/utils"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the current branch to remote (e.g., origin).",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			current := utils.CurrentBranch()
			if current == "" {
				return fmt.Errorf("unable to determine current branch for push")
			}

			logs.Info("Pushing current branch '%s'", current)

			if err := git.PushCurrentBranch(); err != nil {
				logs.Error("Push failed for branch '%s': %v", current, err)
				return err
			}

			fmt.Printf("Branch '%s' pushed to remote.\n", current)
			return nil
		},
	}
}
