package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <share-code>",
		Short: "Pull a shared stack from another user using the provided share code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			code := args[0]
			logs.Info("Pulling shared stack from code '%s'", code)

			err := service.GetCollabService().PullSharedStack(code)
			if err != nil {
				logs.Error("Failed to pull shared stack '%s': %v", code, err)
				return err
			}

			fmt.Printf("Successfully pulled shared stack '%s'.\n", code)
			return nil
		},
	}
}
