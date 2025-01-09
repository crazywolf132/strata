package service

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strata/internal/logs"
	"strata/internal/model"
	"strata/internal/utils"
	"strings"
)

type PRService struct{}

var prSvc *PRService

type prInfo struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Body   string `json:"body"`
	Number int    `json:"number"`
}

// branchPRInfo holds PR information for a branch
type branchPRInfo struct {
	URL   string
	State string
}

func GetPRService() *PRService {
	if prSvc == nil {
		prSvc = &PRService{}
	}
	return prSvc
}

// getBranchPRMap returns a map of branch names to their PR info
func (p *PRService) getBranchPRMap(stack map[string]*model.StackNode) (map[string]branchPRInfo, error) {
	prMap := make(map[string]branchPRInfo)

	// Get all PRs in one call
	cmd := exec.Command("gh", "pr", "list",
		"--json", "url,title,state,body,number,headRefName",
		"--limit", "100", // Increase limit to get all stack PRs
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %v\n%s", err, string(out))
	}

	type extendedPRInfo struct {
		prInfo
		HeadRefName string `json:"headRefName"`
	}

	var prs []extendedPRInfo
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %v", err)
	}

	// Map PRs to their branches
	for _, pr := range prs {
		if _, exists := stack[pr.HeadRefName]; exists {
			prMap[pr.HeadRefName] = branchPRInfo{
				URL:   pr.URL,
				State: pr.State,
			}
		}
	}

	return prMap, nil
}

// generateStackDiagram creates a tree-like representation of the stack with PR links
func (p *PRService) generateStackDiagram(stack map[string]*model.StackNode, currentBranch string) (string, error) {
	var builder strings.Builder
	builder.WriteString("## 🌳 Stack Structure\n\n")

	// Get PR URLs for all branches
	prMap, err := p.getBranchPRMap(stack)
	if err != nil {
		return "", fmt.Errorf("failed to get PR map: %v", err)
	}

	// Find top-level branches (where ParentBranch == "" or parent not in stack)
	topLevels := []string{}
	for br, node := range stack {
		if node.ParentBranch == "" || stack[node.ParentBranch] == nil {
			topLevels = append(topLevels, br)
		}
	}

	visited := make(map[string]bool)
	for i, tl := range topLevels {
		isLast := i == len(topLevels)-1
		p.printStackNode(&builder, stack, stack[tl], 0, visited, currentBranch, prMap, isLast)
	}

	// Add legend
	builder.WriteString("\n### Legend\n")
	builder.WriteString("- 🟢 Ready to merge\n")
	builder.WriteString("- 🟡 In review/draft\n")
	builder.WriteString("- 🔵 Current branch\n")
	builder.WriteString("- ⚪ No PR yet\n")

	return builder.String(), nil
}

func (p *PRService) getStateEmoji(branch string, info branchPRInfo, prBranch string) string {
	if branch == "main" {
		return "🎯" // Special emoji for main branch
	}
	if branch == prBranch {
		return "🔵"
	}
	if info.URL == "" {
		return "⚪"
	}
	switch info.State {
	case "MERGED":
		return "✅"
	case "CLOSED":
		return "❌"
	case "OPEN":
		return "🟢"
	case "DRAFT":
		return "🟡"
	default:
		return "⚪"
	}
}

func (p *PRService) printStackNode(b *strings.Builder, stack map[string]*model.StackNode, node *model.StackNode, level int, visited map[string]bool, prBranch string, prMap map[string]branchPRInfo, isLastChild bool) {
	if node == nil || visited[node.BranchName] {
		return
	}

	// Skip branches without PRs unless they have children with PRs
	hasVisibleChildren := false
	visibleChildren := []string{}
	for _, child := range node.Children {
		if _, hasChildPR := prMap[child]; hasChildPR || child == prBranch {
			hasVisibleChildren = true
			visibleChildren = append(visibleChildren, child)
		} else {
			// Check if child has visible descendants
			if p.hasVisibleDescendants(stack, child, prBranch, prMap, make(map[string]bool)) {
				hasVisibleChildren = true
				visibleChildren = append(visibleChildren, child)
			}
		}
	}

	if !hasVisibleChildren && node.BranchName != prBranch && prMap[node.BranchName].URL == "" && node.BranchName != "main" {
		return
	}

	visited[node.BranchName] = true

	// Create a beautiful tree structure
	var prefix string
	if level > 0 {
		// Build prefix based on parent's visibility
		prefixParts := make([]string, level)
		for i := 0; i < level-1; i++ {
			if i == level-2 && isLastChild {
				prefixParts[i] = "    " // No pipe for last child's parent
			} else {
				prefixParts[i] = "│   " // Pipe for active branches
			}
		}

		// Last connector depends on whether this is the last child
		if isLastChild {
			prefixParts[level-1] = "└── "
		} else {
			prefixParts[level-1] = "├── "
		}

		prefix = strings.Join(prefixParts, "")
	}

	// Get state emoji
	prInfo := prMap[node.BranchName]
	stateEmoji := p.getStateEmoji(node.BranchName, prInfo, prBranch)

	// Format branch name as a link if it has a PR
	branchDisplay := node.BranchName
	if prInfo.URL != "" {
		branchDisplay = fmt.Sprintf("[%s](%s)", node.BranchName, prInfo.URL)
	}

	b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, stateEmoji, branchDisplay))

	// Print visible children in sorted order
	for i, child := range visibleChildren {
		childNode := stack[child]
		if childNode != nil {
			isLast := i == len(visibleChildren)-1
			p.printStackNode(b, stack, childNode, level+1, visited, prBranch, prMap, isLast)
		}
	}
}

// hasVisibleDescendants checks if a branch has any descendants with PRs or is the current branch
func (p *PRService) hasVisibleDescendants(stack map[string]*model.StackNode, branch string, prBranch string, prMap map[string]branchPRInfo, visited map[string]bool) bool {
	if visited[branch] {
		return false
	}
	visited[branch] = true

	node := stack[branch]
	if node == nil {
		return false
	}

	if branch == prBranch || prMap[branch].URL != "" {
		return true
	}

	for _, child := range node.Children {
		if p.hasVisibleDescendants(stack, child, prBranch, prMap, visited) {
			return true
		}
	}

	return false
}

// checkPRExists uses `gh` to check if a PR already exists for the given branch
func (p *PRService) checkPRExists(branch string) (bool, string, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--head", branch,
		"--json", "url,title,state,body,number",
		"--limit", "1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Authentication") {
			logs.Error("GH CLI authentication error: %s", string(out))
			return false, "", fmt.Errorf("gh auth error: %v", err)
		}
		return false, "", fmt.Errorf("failed to check PR existence: %v\n%s", err, string(out))
	}

	// If output is empty array "[]", no PR exists
	if strings.TrimSpace(string(out)) == "[]" {
		return false, "", nil
	}

	// Parse PR info from JSON response
	var prs []prInfo
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return false, "", fmt.Errorf("failed to parse PR info: %v", err)
	}
	if len(prs) == 0 {
		return false, "", nil
	}

	return true, prs[0].URL, nil
}

// updatePRBody updates an existing PR's body with the current stack diagram
func (p *PRService) updatePRBody(branch string, prNumber int, stackDiagram string) error {
	body := fmt.Sprintf("This PR is part of a stacked workflow.\n\n%s", stackDiagram)

	cmd := exec.Command("gh", "pr", "edit",
		fmt.Sprintf("%d", prNumber),
		"--body", body,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update PR body: %v\n%s", err, string(out))
	}
	logs.Info("Updated PR body for '%s' (#%d)", branch, prNumber)
	return nil
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
			if err := p.createOrUpdatePR(br, node.ParentBranch, stack, all); err != nil {
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
	return p.createOrUpdatePR(curr, parent, stack, all)
}

// checkParentPRs verifies that all parent branches (recursively) have PRs
func (p *PRService) checkParentPRs(branch string, stack map[string]*model.StackNode, prMap map[string]branchPRInfo, visited map[string]bool) error {
	if visited[branch] {
		return nil
	}
	visited[branch] = true

	node := stack[branch]
	if node == nil {
		return nil
	}

	// Skip check for main branch
	if node.ParentBranch == "" || node.ParentBranch == "main" {
		return nil
	}

	// Check if parent has a PR
	parentInfo, exists := prMap[node.ParentBranch]
	if !exists || parentInfo.URL == "" {
		return fmt.Errorf("parent branch '%s' does not have a PR yet", node.ParentBranch)
	}

	// Recursively check parent's parents
	return p.checkParentPRs(node.ParentBranch, stack, prMap, visited)
}

func (p *PRService) createOrUpdatePR(branch, base string, stack map[string]*model.StackNode, updateAll bool) error {
	logs.Info("Creating/updating PR for branch '%s' -> base '%s'", branch, base)

	// Get PR URLs for all branches in one call
	prMap, err := p.getBranchPRMap(stack)
	if err != nil {
		return fmt.Errorf("failed to get PR map: %v", err)
	}

	// Check if all parent branches have PRs before proceeding
	if err := p.checkParentPRs(branch, stack, prMap, make(map[string]bool)); err != nil {
		return fmt.Errorf("cannot create PR: %v", err)
	}

	// Generate stack diagram specific to this PR's branch
	stackDiagram, err := p.generateStackDiagram(stack, branch)
	if err != nil {
		return fmt.Errorf("failed to generate stack diagram: %v", err)
	}

	// Use the PR info from the map if it exists
	prInfo, exists := prMap[branch]
	if exists {
		// Get PR number for existing PR
		prs, err := p.getPRInfo(branch)
		if err != nil || len(prs) == 0 {
			return fmt.Errorf("failed to get PR number: %v", err)
		}

		if err := p.updatePRBody(branch, prs[0].Number, stackDiagram); err != nil {
			return err
		}
		fmt.Printf("Updated existing PR for branch '%s': %s\n", branch, prInfo.URL)
	} else {
		// Create new PR
		title := fmt.Sprintf("Strata PR for %s", branch)
		body := fmt.Sprintf("This PR is part of a stacked workflow.\n\n%s", stackDiagram)

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
			if strings.Contains(string(out), "already exists") {
				// This is a race condition - try to update the PR body
				logs.Info("PR was created concurrently for '%s', attempting to update body", branch)
				if prs, err := p.getPRInfo(branch); err == nil && len(prs) > 0 {
					if err := p.updatePRBody(branch, prs[0].Number, stackDiagram); err != nil {
						logs.Warn("Failed to update concurrent PR body: %v", err)
					}
				}
				return nil
			}
			return fmt.Errorf("failed to create PR for '%s': %v\n%s", branch, err, string(out))
		}

		logs.Info("PR created successfully for '%s': %s", branch, string(out))
		fmt.Printf("Created new PR for branch '%s': %s\n", branch, strings.TrimSpace(string(out)))
	}

	// Only update related PRs if updateAll is true
	if updateAll {
		// Update all related PRs in the stack
		for br, info := range prMap {
			if br != branch && info.State != "MERGED" && info.State != "CLOSED" {
				// Generate stack diagram specific to this related PR
				relatedDiagram, err := p.generateStackDiagram(stack, br)
				if err != nil {
					logs.Warn("Failed to generate stack diagram for '%s': %v", br, err)
					continue
				}

				// Get PR number for this branch
				relatedPRs, err := p.getPRInfo(br)
				if err != nil || len(relatedPRs) == 0 {
					logs.Warn("Failed to get PR info for '%s': %v", br, err)
					continue
				}

				// Update the PR body
				if err := p.updatePRBody(br, relatedPRs[0].Number, relatedDiagram); err != nil {
					logs.Warn("Failed to update PR body for '%s': %v", br, err)
					continue
				}
				logs.Info("Updated related PR for '%s'", br)
			}
		}
	}

	return nil
}

// getPRInfo gets detailed PR information for a branch using the cached PR map
func (p *PRService) getPRInfo(branch string) ([]prInfo, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--head", branch,
		"--json", "url,title,state,body,number",
		"--limit", "1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %v\n%s", err, string(out))
	}

	var prs []prInfo
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %v", err)
	}
	return prs, nil
}
