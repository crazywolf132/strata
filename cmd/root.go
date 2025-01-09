package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd is the base command when called without subcommands.
var rootCmd = &cobra.Command{
	Use:   "strata",
	Short: "Strata is a robust, production-ready Git stacking tool.",
	Long: `Strata streamlines stacked PR workflows for GitHub and GitHub Enterprise,
including merges, rebases, collaboration, and offline support—fully tested and production-ready.`,
}

// Execute is called by main.go to run the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		newInitCmd(),
		newDaemonCmd(),
		newAddCmd(),
		newRenameCmd(),
		newMergeCmd(),
		newUpdateCmd(),
		newPrCmd(),
		newShareCmd(),
		newUseCmd(),
		newViewCmd(),
		newConfigCmd(),
		newHookCmd(),
		newPushCmd(),
		newRebaseCmd(),
		newServerCmd(),
		newCICmd(),
	)
}
