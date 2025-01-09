package cmd

import (
	"fmt"
	"strata/internal/locks"
	"strata/internal/service"

	"github.com/spf13/cobra"
)

func newCICmd() *cobra.Command {
	ciCmd := &cobra.Command{
		Use:   "ci",
		Short: "CI/CD related commands for Strata (checks if a branch can be merged, etc.)",
	}

	checkCmd := &cobra.Command{
		Use:   "check <branch>",
		Short: "Check if <branch> can be merged based on the stack state.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			branch := args[0]
			err := service.GetCIService().CheckMergeFeasibility(branch)
			if err != nil {
				fmt.Println("CI check failed:", err)
				// return an error so the pipeline can fail
				return err
			}
			fmt.Printf("Branch '%s' can be safely merged.\n", branch)
			return nil
		},
	}

	ciCmd.AddCommand(checkCmd)
	return ciCmd
}
