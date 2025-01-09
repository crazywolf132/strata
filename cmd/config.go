package cmd

import (
	"fmt"
	"strata/internal/config"
	"strata/internal/locks"
	"strata/internal/logs"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Strata configuration (local or global).",
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value (local overrides global).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := config.GetConfigValue(key)
			fmt.Printf("%s = %s\n", key, val)
			return nil
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a local config value.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			locks.LockRepo()
			defer locks.UnlockRepo()

			key := args[0]
			value := args[1]
			if err := config.SetConfigValue(key, value, false); err != nil {
				logs.Error("Failed to set local config '%s': %v", key, err)
				return err
			}
			fmt.Printf("Set local config: %s = %s\n", key, value)
			return nil
		},
	}

	setGlobalCmd := &cobra.Command{
		Use:   "set-global <key> <value>",
		Short: "Set a global config value in ~/.strata/global_config.yaml.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			if err := config.SetConfigValue(key, value, true); err != nil {
				logs.Error("Failed to set global config '%s': %v", key, err)
				return err
			}
			fmt.Printf("Set global config: %s = %s\n", key, value)
			return nil
		},
	}

	cfgCmd.AddCommand(getCmd, setCmd, setGlobalCmd)
	return cfgCmd
}
