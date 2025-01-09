package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strata/internal/config"
	"strata/internal/logs"
	"strings"
	"time"
)

// We wrap core Git commands with robust error checks and partial rollback if needed.

func IsGitRepo() bool {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return false
	}
	return true
}

func CheckoutNewBranch(branchName string) error {
	// Ensure there's no uncommitted changes
	if err := ensureCleanWorkingTree(); err != nil {
		return err
	}

	cmd := exec.Command("git", "checkout", "-b", branchName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b %s failed: %v\n%s", branchName, err, string(out))
	}
	return nil
}

func RenameBranch(oldName, newName string) error {
	if err := ensureCleanWorkingTree(); err != nil {
		return err
	}
	// Attempt local rename
	cmd := exec.Command("git", "branch", "-m", oldName, newName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("branch rename error: %v\n%s", err, string(out))
	}
	// Attempt remote rename (push :old new)
	cmd = exec.Command("git", "push", "origin", ":"+oldName, newName)
	out, err = cmd.CombinedOutput()
	if err != nil {
		logs.Warn("Remote rename might have failed (possibly no remote branch). Details: %s", string(out))
	}
	return nil
}

func MergeBranch(src, target string) error {
	// Create a save point
	txTag := CreateTxTag("merge")
	defer CleanupTxTag(txTag)

	// checkout target
	if err := checkoutBranch(target); err != nil {
		RevertToTag(txTag)
		return err
	}

	cmd := exec.Command("git", "merge", "--no-ff", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exec.Command("git", "merge", "--abort").Run()
		RevertToTag(txTag)
		return fmt.Errorf("merge %s -> %s failed: %v\n%s", src, target, err, string(out))
	}
	return nil
}

// CreateTxTag creates a temporary tag like `strata-tx-<prefix>-<timestamp>`
func CreateTxTag(prefix string) string {
	t := time.Now().UnixNano()
	tagName := fmt.Sprintf("strata-tx-%s-%d", prefix, t)
	exec.Command("git", "tag", tagName).Run() // Ignoring error
	return tagName
}

// RevertToTag reverts HEAD to the specified tag
func RevertToTag(tag string) {
	// revert HEAD to that tag
	cmd := exec.Command("git", "reset", "--hard", tag)
	cmd.Run() // ignore errors (we do best effort)
}

// CleanupTxTag removes the specified transaction tag
func CleanupTxTag(tag string) {
	// remove the tag
	exec.Command("git", "tag", "-d", tag).Run()
}

func checkoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout branch '%s' error: %v\n%s", branch, err, string(out))
	}
	return nil
}

func PushCurrentBranch() error {
	if err := ensureCleanWorkingTree(); err != nil {
		// We allow pushing with uncommitted changes in Git, but let's be strict here to avoid partial pushes
		return fmt.Errorf("cannot push with uncommitted changes: %v", err)
	}
	cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push error: %v\n%s", err, string(out))
	}
	return nil
}

// RebaseBranch performs an interactive rebase with fallback to manual conflict resolution prompt
func RebaseBranch(branch, onto string) error {
	// Create a save point
	txTag := CreateTxTag("rebase")
	defer CleanupTxTag(txTag)

	if err := ensureCleanWorkingTree(); err != nil {
		RevertToTag(txTag)
		return err
	}
	// checkout the target branch
	if err := checkoutBranch(branch); err != nil {
		RevertToTag(txTag)
		return err
	}

	cmd := exec.Command("git", "rebase", onto)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "CONFLICT") {
			// Handle conflict
			cErr := handleRebaseConflict()
			if cErr != nil {
				// user might abort
				RevertToTag(txTag)
				return cErr
			}
			// If the user successfully continues, it's presumably fine
			return nil
		}
		// general fail
		exec.Command("git", "rebase", "--abort").Run()
		RevertToTag(txTag)
		return fmt.Errorf("rebase %s onto %s failed: %v\n%s", branch, onto, err, string(out))
	}
	return nil
}

func handleRebaseConflict() error {
	policy := config.GetConfigValue("auto_conflict_resolution")
	switch policy {
	case "ours":
		// automatically choose ours for conflicting files
		exec.Command("git", "checkout", "--ours", ".").Run()
		exec.Command("git", "add", ".").Run()
		exec.Command("git", "rebase", "--continue").Run()
		// we'd still check if more conflicts remain.
		return nil
	case "theirs":
		exec.Command("git", "checkout", "--theirs", ".").Run()
		exec.Command("git", "add", ".").Run()
		exec.Command("git", "rebase", "--continue").Run()
		return nil
	default:
		return handleRebaseConflictManually()
	}
}

// handleRebaseConflictManually prompts user to manually fix conflicts, then continue or abort
func handleRebaseConflictManually() error {
	fmt.Println("Rebase conflict detected. Please resolve conflicts in your editor.")
	for {
		fmt.Print("Type 'continue' when conflicts are resolved, or 'abort' to cancel rebase: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return fmt.Errorf("no input; rebase cannot proceed")
		}
		ans := scanner.Text()
		switch ans {
		case "continue":
			cmd := exec.Command("git", "rebase", "--continue")
			out, err := cmd.CombinedOutput()
			if err != nil {
				if strings.Contains(string(out), "CONFLICT") {
					fmt.Println("Still conflicts remain. Please resolve and type 'continue' again.")
					continue
				} else {
					return fmt.Errorf("rebase --continue failed: %v\n%s", err, string(out))
				}
			}
			// success
			return nil
		case "abort":
			exec.Command("git", "rebase", "--abort").Run()
			return fmt.Errorf("rebase aborted by user")
		default:
			fmt.Println("Unknown input. Type 'continue' or 'abort'.")
		}
	}
}

// ensureCleanWorkingTree checks for uncommitted changes
func ensureCleanWorkingTree() error {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v\n%s", err, string(out))
	}
	status := strings.TrimSpace(string(out))
	if status != "" {
		return fmt.Errorf("working tree not clean; commit or stash changes first:\n%s", status)
	}
	return nil
}

// PullBranch merges remote changes into the current branch
func PullBranch() error {
	cmd := exec.Command("git", "pull", "--rebase")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "CONFLICT") {
			// handle similarly to handleRebaseConflict
			if e := handleRebaseConflict(); e != nil {
				return e
			}
			return nil
		}
		return fmt.Errorf("git pull --rebase failed: %v\n%s", err, string(out))
	}
	return nil
}

// Optional auto-fetch mechanism
func FetchAll() error {
	cmd := exec.Command("git", "fetch", "--all")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch error: %v\n%s", err, string(out))
	}
	return nil
}

func SyncWithRemote(branch string) error {
	// checkout branch
	if err := checkoutBranch(branch); err != nil {
		return err
	}
	// fetch + pull
	if err := FetchAll(); err != nil {
		return err
	}
	if err := PullBranch(); err != nil {
		return err
	}
	return nil
}

// Stash/unstash might be used if we want to preserve user changes during certain operations
func StashSave() error {
	cmd := exec.Command("git", "stash", "push", "-u", "-m", "Strata-auto-stash")
	_, err := cmd.CombinedOutput()
	return err
}

func StashPop() error {
	cmd := exec.Command("git", "stash", "pop")
	_, err := cmd.CombinedOutput()
	return err
}

// Some operations might want to use a time-based tag or commit. We can do that if needed.
func TagCommit(tagName, message string) error {
	cmd := exec.Command("git", "tag", "-a", tagName, "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git tag error: %v\n%s", err, string(out))
	}
	return nil
}

// E.g., to revert partially merged changes on error
func RevertToCommit(commitHash string) error {
	cmd := exec.Command("git", "reset", "--hard", commitHash)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to revert to %s: %v\n%s", commitHash, err, string(out))
	}
	return nil
}

// GetGitDir returns the absolute path to the .git directory
func GetGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to find .git directory: %v\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// IsBranchMergedUpstream checks if a branch has been merged into its upstream branch
func IsBranchMergedUpstream(branch string) (bool, error) {
	// First, fetch to ensure we have latest remote info
	if err := FetchAll(); err != nil {
		return false, err
	}

	// Check if branch exists on remote
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branch)
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) == 0 {
		// Branch doesn't exist on remote, so can't be merged
		return false, nil
	}

	// Get the merge-base (common ancestor) with origin/HEAD
	cmd = exec.Command("git", "merge-base", branch, "origin/HEAD")
	mergeBase, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to find merge-base: %v\n%s", err, string(mergeBase))
	}

	// Get the latest commit of the branch
	cmd = exec.Command("git", "rev-parse", branch)
	branchCommit, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to get branch commit: %v\n%s", err, string(branchCommit))
	}

	// If merge-base equals branch commit, the branch is merged
	return strings.TrimSpace(string(mergeBase)) == strings.TrimSpace(string(branchCommit)), nil
}
