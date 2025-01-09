package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/locks"
	"strata/internal/logs"
	"strata/internal/service"
)

func newShareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "share",
		Short: "Generate a share code so another user can clone your stack locally.",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			code, err := service.GetCollabService().GenerateShareCode()
			if err != nil {
				logs.Error("Failed to generate share code: %v", err)
				return err
			}

			fmt.Printf("Your stack share code: %s\n", code)
			logs.Info("Generated share code: %s", code)
			return nil
		},
	}
}
