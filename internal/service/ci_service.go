package service

import (
	"fmt"
	"strata/internal/logs"
)

type CIService struct{}

var ciSvc *CIService

func GetCIService() *CIService {
	if ciSvc == nil {
		ciSvc = &CIService{}
	}
	return ciSvc
}

// CheckMergeFeasibility ensures the parent is merged, no conflicts remain etc.
// This is a simplistic exampl. Real logic might check if parent is fully merged, or if there's a PR conflict, etc.
func (c *CIService) CheckMergeFeasibility(branch string) error {
	s := GetStackService()
	st := s.GetStack()
	node, ok := st[branch]
	if !ok {
		return fmt.Errorf("branch '%s' not found in stack", branch)
	}

	// If parent is not in the stack (like 'main'), we assume it's always feasible
	if node.ParentBranch == "" {
		logs.Info("'%s' has no parent in the stack. Possibly top-level. Marking feasible.", branch)
		return nil
	}

	// Check if parent is in the stack. If the parent's node is still open (not merged), we can't merge child yet.
	parentNode, pOk := st[node.ParentBranch]
	if pOk {
		// if parent is still in the stack, it means it hasn't fully merged.
		return fmt.Errorf("parent branch '%s' not yet merged, so '%s' cannot be merged in the stack", parentNode.BranchName, branch)
	}

	// We might also check for rebase conflicts, etc. byt that's more advanced.
	return nil
}
