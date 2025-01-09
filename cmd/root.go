package cmd

import (
	"strata/internal/logs"
	"strata/internal/ui"

	"github.com/spf13/cobra"
)

var (
	verbose bool
)

// rootCmd is the base command when called without subcommands.
var rootCmd = &cobra.Command{
	Use:   "strata",
	Short: "Strata is a robust, production-ready Git stacking tool.",
	Long: `Strata streamlines stacked PR workflows for GitHub and GitHub Enterprise,
including merges, rebases, collaboration, and offline support—fully tested and production-ready.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logs.SetVerbose(verbose)
		if err := logs.InitLogger(); err != nil {
			return err
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logs.Close()
	},
}

// Execute is called by main.go to run the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

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

	rootCmd.SetUsageTemplate(ui.ColorHeadings(rootCmd.UsageTemplate()))
}
