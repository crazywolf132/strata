package service

import (
	"fmt"
	"strata/internal/git"
	"strata/internal/logs"
)

type RebaseService struct{}

var rbService *RebaseService

func GetRebaseService() *RebaseService {
	if rbService == nil {
		rbService = &RebaseService{}
	}
	return rbService
}

// RebaseBranch handles a direct rebase of <branch> onto <onto>
func (r *RebaseService) RebaseBranch(branch, onto string) error {
	logs.Info("Performing direct rebase of '%s' onto '%s'", branch, onto)
	if err := git.RebaseBranch(branch, onto); err != nil {
		return fmt.Errorf("reabse failed: %v", err)
	}
	logs.Info("Rebase of '%s' onto '%s' completed successfully.", branch, onto)
	return nil
}
