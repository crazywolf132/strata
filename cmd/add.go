package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <branch-name>",
		Short: "Create a new layer (branch) on top of the current branch.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			branchName := args[0]
			logs.Info("Creating new stack layer: %s", branchName)

			err := service.GetStackService().CreateNewLayer(branchName)
			if err != nil {
				logs.Error("Failed to create new layer '%s': %v", branchName, err)
				return err
			}

			fmt.Printf("New layer '%s' created successfully and checked out.\n", branchName)
			return nil
		},
	}
}
