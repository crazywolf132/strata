package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a stack layer locally and on remote, updating the stack tree.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			oldName := args[0]
			newName := args[1]

			logs.Info("Renaming branch '%s' to '%s'", oldName, newName)

			err := service.GetStackService().RenameLayer(oldName, newName)
			if err != nil {
				logs.Error("Rename failed from '%s' to '%s': %v", oldName, newName, err)
				return err
			}

			fmt.Printf("Branch '%s' renamed to '%s' successfully.\n", oldName, newName)
			return nil
		},
	}
}
