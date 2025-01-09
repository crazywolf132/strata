package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View the current stack in a tree-like format.",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			tree, err := service.GetStackService().ViewStackTree()
			if err != nil {
				logs.Error("Failed to view stack tree: %v", err)
				return err
			}

			fmt.Println(tree)
			return nil
		},
	}
}
