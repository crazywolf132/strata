package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newRebaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rebase <branch-name> <onto-branch>",
		Short: "Rebase <branch> onto <onto-branch> with conflict handling.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			locks.LockRepo()
			defer locks.UnlockRepo()

			branch := args[0]
			onto := args[1]

			logs.Info("Rebasing branch '%s' onto '%s'", branch, onto)
			err := service.GetRebaseService().RebaseBranch(branch, onto)
			if err != nil {
				logs.Error("Rebase failed for '%s' onto '%s': %v", branch, onto, err)
				return err
			}

			fmt.Printf("Branch '%s' succesfully rebased onto '%s'.\n", branch, onto)
			return nil

			return nil
		},
	}
}
