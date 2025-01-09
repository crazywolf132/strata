package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <branch-name>",
		Short: "Merge a stack layer into its parent (or main if no parent).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			branch := args[0]
			logs.Info("Merging branch '%s'", branch)

			err := service.GetStackService().MergeLayer(branch)
			if err != nil {
				logs.Error("Failed to merge branch '%s': %v", branch, err)
				return err
			}

			fmt.Printf("Branch '%s' merged successfully.\n", branch)
			return nil
		},
	}
}
