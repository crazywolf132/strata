package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update the entire stack by rebasing or merging each branch on its parent.",
		Long: `Attempts to bring all branches up-to-date with their parents. 
Ensures minimal conflicts and offers interactive resolution if needed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			logs.Info("Updating entire stack via rebase/merge strategy...")
			err := service.GetStackService().UpdateEntireStack()
			if err != nil {
				logs.Error("Update failed: %v", err)
				return err
			}

			fmt.Println("Entire stack updated successfully.")

			return nil
		},
	}
}
