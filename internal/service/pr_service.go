package service

import (
	"fmt"
	"os/exec"
	"strata/internal/logs"
	"strata/internal/utils"
	"strings"
)

type PRService struct{}

var prSvc *PRService

func GetPRService() *PRService {
	if prSvc == nil {
		prSvc = &PRService{}
	}
	return prSvc
}

// CreatePR uses `gh` to open PR(s). If all==true, open for every unmerged stack branch.
func (p *PRService) CreatePR(all bool) error {
	s := GetStackService()
	stack := s.GetStack()

	if all {
		// Open PR for each branch that has a parent
		for br, node := range stack {
			if node.ParentBranch == "" {
				continue
			}
			if err := createSinglePR(br, node.ParentBranch); err != nil {
				return err
			}
		}
		return nil
	}

	// otherwise, just for current branch
	curr := utils.CurrentBranch()
	if curr == "" {
		return fmt.Errorf("cannot determine current branch to create PR")
	}
	parent := "main" // default
	if node, ok := stack[curr]; ok && node.ParentBranch != "" {
		parent = node.ParentBranch // use parent branch if exists
	}
	return createSinglePR(curr, parent)
}

func createSinglePR(branch, base string) error {
	logs.Info("Creating PR for branch '%s' -> base '%s'", branch, base)

	title := fmt.Sprintf("Strata PR for %s", branch)
	body := "This PR is part of a stacked workflow."

	cmd := exec.Command("gh", "pr", "create",
		"--base", base,
		"--head", branch,
		"--title", title,
		"--body", body,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Authentication") {
			logs.Error("GH CLI authentication error: %s", string(out))
			return fmt.Errorf("gh auth error: %v", err)
		}
		return fmt.Errorf("failed to create PR for '%s': %v\n%s", branch, err, string(out))
	}
	logs.Info("PR created successfully for '%s': %s", branch, string(out))
	return nil
}
