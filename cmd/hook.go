package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"strata/internal/hooks"
	"strata/internal/locks"
	"strata/internal/logs"
)

func newHookCmd() *cobra.Command {
	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage or list custom Strata hooks (scripts).",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all hooks configured in this repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			hs := hooks.ListHooks()
			if len(hs) == 0 {
				fmt.Println("No hooks configured.")
				return nil
			}
			fmt.Println("Configured hooks:")
			for _, h := range hs {
				fmt.Println(" -", h)
			}
			return nil
		},
	}

	addCmd := &cobra.Command{
		Use:   "add <event> <script-path>",
		Short: "Add a new hook that runs on the specified event.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			event := args[0]
			script := args[1]
			err := hooks.AddHook(event, script)
			if err != nil {
				logs.Error("Failed to add hook for event '%s': %v", event, err)
				return err
			}
			fmt.Printf("Hook added for event '%s' -> script '%s'\n", event, script)
			return nil
		},
	}

	hookCmd.AddCommand(listCmd, addCmd)
	return hookCmd
}
