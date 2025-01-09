package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newPrCmd() *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage GitHub pull requests for your stack",
		Long: `Create or update pull requests on GitHub / GitHub Enterprise for 
individual stack layers or for all unmerged layers in the stack.`,
	}

	createCmd := &cobra.Command{
		Use:   "create [--all]",
		Short: "Create PR(s) for the current or all stacked branches.",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			all, _ := cmd.Flags().GetBool("all")
			logs.Info("Creating PR(s) on GitHub (all=%v)", all)

			err := service.GetPRService().CreatePR(all)
			if err != nil {
				logs.Error("Failed to create PR(s): %v", err)
				return err
			}

			fmt.Println("PR(s) created successfully on GitHub.")
			return nil
		},
	}
	createCmd.Flags().Bool("all", false, "Create PRs for all unmerged branches")

	prCmd.AddCommand(createCmd)
	return prCmd
}
