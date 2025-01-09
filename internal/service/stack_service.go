package service

import (
	"fmt"
	"os/exec"
	"strata/internal/git"
	"strata/internal/hooks"
	"strata/internal/logs"
	"strata/internal/model"
	"strata/internal/store"
	"strata/internal/utils"
	"strings"
	"time"
)

type StackService struct {
	stack model.StackTree
}

var stackSvc *StackService

func GetStackService() *StackService {
	if stackSvc == nil {
		st, err := store.LoadStack()
		if err != nil {
			logs.Error("Failed to load stack from disk: %v", err)
			st = model.StackTree{}
		}
		stackSvc = &StackService{stack: st}
	}
	return stackSvc
}

func (s *StackService) CreateNewLayer(branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	current := utils.CurrentBranch()
	if current == "" {
		return fmt.Errorf("cannot determine current branch to stack on")
	}

	if err := git.CheckoutNewBranch(branchName); err != nil {
		return err
	}

	// Update in-memory stack
	node := &model.StackNode{
		BranchName:   branchName,
		ParentBranch: current,
		Children:     []string{},
		CreatedBy:    utils.GetGithubUsername(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.stack[branchName] = node

	// Add child to parent's node
	if parent, ok := s.stack[current]; ok {
		parent.Children = append(parent.Children, branchName)
	} else {
		s.stack[current] = &model.StackNode{
			BranchName: current,
			Children:   []string{branchName},
		}
	}

	if err := store.SaveStack(s.stack); err != nil {
		return err
	}

	hooks.RunHooks("createLayer", branchName)
	return nil
}

func (s *StackService) RenameLayer(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("invalid rename: old or new name empty")
	}

	if _, ok := s.stack[oldName]; !ok {
		return fmt.Errorf("branch '%s' not found in stack", oldName)
	}

	if err := git.RenameBranch(oldName, newName); err != nil {
		return err
	}

	node := s.stack[oldName]
	node.BranchName = newName
	node.UpdatedAt = time.Now()
	s.stack[newName] = node
	delete(s.stack, oldName)

	// Update references in parent's Children
	for _, nd := range s.stack {
		for i, c := range nd.Children {
			if c == oldName {
				nd.Children[i] = newName
			}
		}
	}

	if err := store.SaveStack(s.stack); err != nil {
		return err
	}
	hooks.RunHooks("renameLayer", newName)
	return nil
}

func (s *StackService) MergeLayer(branch string) error {
	node, exists := s.stack[branch]
	if !exists {
		return fmt.Errorf("branch '%s' not in stack", branch)
	}

	parent := node.ParentBranch
	if parent == "" {
		// If no parent, assume main
		parent = "main"
	}
	logs.Info("Merging '%s' into '%s'", branch, parent)

	if err := git.MergeBranch(branch, parent); err != nil {
		return err
	}

	// Remove from parent's children
	if parentNode, ok := s.stack[parent]; ok {
		newKids := []string{}
		for _, c := range parentNode.Children {
			if c != branch {
				newKids = append(newKids, c)
			}
		}
		parentNode.Children = newKids
	}
	delete(s.stack, branch)

	if err := store.SaveStack(s.stack); err != nil {
		return err
	}
	hooks.RunHooks("mergeLayer", branch)
	return nil
}

// UpdateEntireStack attempts to rebase each child on its parent, topologically
func (s *StackService) UpdateEntireStack() error {
	// Create a save point for the entire update operation
	txTag := git.CreateTxTag("stack-update")
	defer git.CleanupTxTag(txTag)

	// We'll do a topological sort: first update branches whose parents are up to date.
	updated := map[string]bool{}
	toDelete := []string{} // Track branches to remove

	for {
		progressed := false
		for br, node := range s.stack {
			if updated[br] {
				continue
			}
			p := node.ParentBranch
			if p == "" {
				// treat as top-level, might be main or something else
				// Try to sync with remote (optional)
				if err := git.SyncWithRemote(br); err != nil {
					logs.Warn("Sync with remote for top-level '%s' failed: %v", br, err)
				}
				updated[br] = true
				progressed = true
			} else {
				// Check if branch is merged upstream
				isMerged, err := git.IsBranchMergedUpstream(br)
				if err != nil {
					logs.Warn("Failed to check if '%s' is merged: %v", br, err)
				} else if isMerged {
					logs.Info("Branch '%s' has been merged upstream, will be removed from stack", br)
					toDelete = append(toDelete, br)
					updated[br] = true
					progressed = true
					continue
				}

				// only proceed if parent is updated
				if updated[p] {
					// rebase br onto p
					logs.Info("Rebasing '%s' onto '%s' during stack update...", br, p)
					if err := git.RebaseBranch(br, p); err != nil {
						git.RevertToTag(txTag)
						return fmt.Errorf("rebase failed for '%s': %v", br, err)
					}

					// optionally push br
					if e2 := git.PushCurrentBranch(); e2 != nil {
						logs.Warn("push after rebase failed for '%s': %v", br, e2)
					}
					// Updating timestamps
					node.UpdatedAt = time.Now()
					updated[br] = true
					progressed = true
				}
			}
		}
		if !progressed {
			break
		}
	}

	// Process merged branches in order (parents before children)
	for len(toDelete) > 0 {
		br := toDelete[0]
		toDelete = toDelete[1:]
		node := s.stack[br]
		if node == nil {
			continue
		}

		// Create a save point for this branch removal operation
		branchTxTag := git.CreateTxTag(fmt.Sprintf("remove-%s", br))

		parent := node.ParentBranch
		if parent == "" {
			parent = "main" // Default to main if no parent
		}

		// First rebase all children onto the parent
		for _, child := range node.Children {
			childNode, exists := s.stack[child]
			if !exists {
				continue
			}

			logs.Info("Rebasing child '%s' onto parent '%s' before removing merged branch '%s'", child, parent, br)
			if err := git.RebaseBranch(child, parent); err != nil {
				logs.Error("Failed to rebase child '%s' onto '%s': %v", child, parent, err)
				git.RevertToTag(branchTxTag)
				git.CleanupTxTag(branchTxTag)
				continue
			}

			// Update child's parent reference
			childNode.ParentBranch = parent
			childNode.UpdatedAt = time.Now()

			// Add child to parent's children list
			if parentNode, ok := s.stack[parent]; ok {
				parentNode.Children = append(parentNode.Children, child)
			}
		}

		// Remove the branch from its parent's children list
		if parentNode, ok := s.stack[parent]; ok {
			newChildren := make([]string, 0, len(parentNode.Children))
			for _, child := range parentNode.Children {
				if child != br {
					newChildren = append(newChildren, child)
				}
			}
			parentNode.Children = newChildren
		}

		// Delete the branch locally if it exists
		cmd := exec.Command("git", "branch", "-D", br)
		if err := cmd.Run(); err != nil {
			logs.Warn("Failed to delete local branch '%s': %v", br, err)
		}

		// Remove the branch from the stack
		delete(s.stack, br)
		logs.Info("Removed merged branch '%s' from stack", br)

		// Save after each branch removal to ensure we don't lose state
		if err := store.SaveStack(s.stack); err != nil {
			git.RevertToTag(branchTxTag)
			git.CleanupTxTag(branchTxTag)
			return fmt.Errorf("failed to save stack after removing branch '%s': %v", br, err)
		}

		git.CleanupTxTag(branchTxTag)
	}

	hooks.RunHooks("updateStack", "")
	return nil
}

func (s *StackService) ViewStackTree() (string, error) {
	// Render a tree from top-level branches
	var builder strings.Builder
	visited := map[string]bool{}

	// find top-level branches (where ParentBranch == "" or parent not in stack)
	topLevels := []string{}
	for br, node := range s.stack {
		if node.ParentBranch == "" || s.stack[node.ParentBranch] == nil {
			topLevels = append(topLevels, br)
		}
	}

	for _, tl := range topLevels {
		printNode(&builder, s.stack, s.stack[tl], 0, visited)
	}
	return builder.String(), nil
}

func printNode(b *strings.Builder, st model.StackTree, node *model.StackNode, level int, visited map[string]bool) {
	if node == nil || visited[node.BranchName] {
		return
	}

	visited[node.BranchName] = true
	indent := strings.Repeat("  ", level)
	b.WriteString(fmt.Sprintf("%s- %s\n", indent, node.BranchName))
	for _, child := range node.Children {
		printNode(b, st, st[child], level+1, visited)
	}
}

// This helper is for tests or advanced flows where we might want to reload the stack.
func (s *StackService) ReloadStack() error {
	st, err := store.LoadStack()
	if err != nil {
		return err
	}
	s.stack = st
	return nil
}

func (s *StackService) GetStack() model.StackTree {
	return s.stack
}
