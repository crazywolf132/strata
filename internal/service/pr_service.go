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

func GetPRService() *PRService {
	if prSvc == nil {
		prSvc = &PRService{}
	}
	return prSvc
}

// getBranchPRMap returns a map of branch names to their PR URLs
func (p *PRService) getBranchPRMap(stack map[string]*model.StackNode) (map[string]string, error) {
	prMap := make(map[string]string)
	for br := range stack {
		exists, url, err := p.checkPRExists(br)
		if err != nil {
			logs.Warn("Failed to get PR info for '%s': %v", br, err)
			continue
		}
		if exists {
			prMap[br] = url
		}
	}
	return prMap, nil
}

// generateStackDiagram creates a tree-like representation of the stack with PR links
func (p *PRService) generateStackDiagram(stack map[string]*model.StackNode, currentBranch string) (string, error) {
	var builder strings.Builder
	builder.WriteString("## Stack Structure\n\n")

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
	for _, tl := range topLevels {
		p.printStackNode(&builder, stack, stack[tl], 0, visited, currentBranch, prMap)
	}

	return builder.String(), nil
}

func (p *PRService) printStackNode(b *strings.Builder, stack map[string]*model.StackNode, node *model.StackNode, level int, visited map[string]bool, currentBranch string, prMap map[string]string) {
	if node == nil || visited[node.BranchName] {
		return
	}

	// Skip branches without PRs unless they have children with PRs
	hasVisibleChildren := false
	for _, child := range node.Children {
		if _, hasChildPR := prMap[child]; hasChildPR {
			hasVisibleChildren = true
			break
		}
	}

	if !hasVisibleChildren && node.BranchName != currentBranch && prMap[node.BranchName] == "" {
		return
	}

	visited[node.BranchName] = true
	indent := strings.Repeat("  ", level)
	pointer := ""
	if node.BranchName == currentBranch {
		pointer = " 👉" // Point to current branch
	}

	// Format branch name as a link if it has a PR
	branchDisplay := node.BranchName
	if url, hasPR := prMap[node.BranchName]; hasPR {
		branchDisplay = fmt.Sprintf("[%s](%s)", node.BranchName, url)
	}

	b.WriteString(fmt.Sprintf("%s- %s%s\n", indent, branchDisplay, pointer))

	for _, child := range node.Children {
		p.printStackNode(b, stack, stack[child], level+1, visited, currentBranch, prMap)
	}
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

	// First, generate stack diagram that will be used in PR bodies
	stackDiagram, err := p.generateStackDiagram(stack, utils.CurrentBranch())
	if err != nil {
		return fmt.Errorf("failed to generate stack diagram: %v", err)
	}

	if all {
		// Open PR for each branch that has a parent
		for br, node := range stack {
			if node.ParentBranch == "" {
				continue
			}
			if err := p.createOrUpdatePR(br, node.ParentBranch, stackDiagram); err != nil {
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
	return p.createOrUpdatePR(curr, parent, stackDiagram)
}

func (p *PRService) createOrUpdatePR(branch, base, stackDiagram string) error {
	logs.Info("Creating/updating PR for branch '%s' -> base '%s'", branch, base)

	// Check if PR already exists
	cmd := exec.Command("gh", "pr", "list",
		"--head", branch,
		"--json", "url,title,state,body,number",
		"--limit", "1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Authentication") {
			logs.Error("GH CLI authentication error: %s", string(out))
			return fmt.Errorf("gh auth error: %v", err)
		}
		return fmt.Errorf("failed to check PR existence: %v\n%s", err, string(out))
	}

	var prs []prInfo
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return fmt.Errorf("failed to parse PR info: %v", err)
	}

	// If PR exists, update its body
	if len(prs) > 0 {
		pr := prs[0]
		if err := p.updatePRBody(branch, pr.Number, stackDiagram); err != nil {
			return err
		}
		fmt.Printf("Updated existing PR for branch '%s': %s\n", branch, pr.URL)
		return nil
	}

	// Create new PR
	title := fmt.Sprintf("Strata PR for %s", branch)
	body := fmt.Sprintf("This PR is part of a stacked workflow.\n\n%s", stackDiagram)

	cmd = exec.Command("gh", "pr", "create",
		"--base", base,
		"--head", branch,
		"--title", title,
		"--body", body,
	)

	out, err = cmd.CombinedOutput()
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
	return nil
}

// getPRInfo gets detailed PR information for a branch
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
