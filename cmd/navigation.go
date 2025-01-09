package cmd

import (
	"fmt"
	"os/exec"
	"strata/internal/service"
	"strata/internal/utils"

	"github.com/spf13/cobra"
)

func newNextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Switch to the next branch in the stack",
		Long:  "Switch to the next branch in the stack (child branch)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := service.GetStackService()
			stack := s.GetStack()

			curr := utils.CurrentBranch()
			if curr == "" {
				return fmt.Errorf("cannot determine current branch")
			}

			node, ok := stack[curr]
			if !ok {
				return fmt.Errorf("current branch '%s' not found in stack", curr)
			}

			if len(node.Children) == 0 {
				return fmt.Errorf("no next branch found - '%s' has no children", curr)
			}

			// For now, just take the first child
			nextBranch := node.Children[0]

			// Execute git checkout
			gitCmd := exec.Command("git", "checkout", nextBranch)
			if out, err := gitCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to checkout branch '%s': %v\n%s", nextBranch, err, string(out))
			}

			fmt.Printf("Switched to branch '%s'\n", nextBranch)
			return nil
		},
	}
	return cmd
}

func newPrevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prev",
		Short: "Switch to the previous branch in the stack",
		Long:  "Switch to the previous branch in the stack (parent branch)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := service.GetStackService()
			stack := s.GetStack()

			curr := utils.CurrentBranch()
			if curr == "" {
				return fmt.Errorf("cannot determine current branch")
			}

			node, ok := stack[curr]
			if !ok {
				return fmt.Errorf("current branch '%s' not found in stack", curr)
			}

			if node.ParentBranch == "" {
				return fmt.Errorf("no previous branch found - '%s' is at the root", curr)
			}

			// Execute git checkout
			gitCmd := exec.Command("git", "checkout", node.ParentBranch)
			if out, err := gitCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to checkout branch '%s': %v\n%s", node.ParentBranch, err, string(out))
			}

			fmt.Printf("Switched to branch '%s'\n", node.ParentBranch)
			return nil
		},
	}
	return cmd
}
