package utils

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"strata/internal/logs"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// CurrentBranch returns the current Git branch, or empty if error
func CurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// RandomShareCode returns a short random code for stack sharing.
func RandomShareCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// GetGithubUsername attempts to determine the GitHub username of the current user.
// 1. "gh api user -q .login"
// 2. fallback: "git config user.name"
// 3. fallback: $USER or $USERNAME
// 4. fallback: "anonymous"
func GetGithubUsername() string {
	// 1. Try "gh api user"
	username := tryGHApiUser()
	if username != "" {
		return username
	}

	// 2. Fallback: "git config user.name"
	username = tryGitConfigUserName()
	if username != "" {
		return username
	}

	// 3. Fallback: environment variables
	userEnv := os.Getenv("USER")
	if userEnv == "" {
		userEnv = os.Getenv("USERNAME") // Windows
	}
	if userEnv != "" {
		logs.Info("Falling back to $USER/$USERNAME = %s", userEnv)
		return userEnv
	}

	// 4. Final fallback
	logs.Warn("Unable to determine GitHub or local username. Using 'anonymous'.")
	return "anonymous"
}

// tryGHApiUser uses "gh api user -q .login" to retrieve the current GH username.
func tryGHApiUser() string {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		logs.Debug("gh api user failed: %v", err)
		return ""
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return ""
	}
	logs.Debug("Detected GitHub username via gh: %s", out)
	return out
}

// tryGitConfigUserName runs "git config user.name" to retrieve the user’s Git name.
func tryGitConfigUserName() string {
	cmd := exec.Command("git", "config", "user.name")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		logs.Debug("git config user.name failed: %v", err)
		return ""
	}
	out := strings.TrimSpace(stdout.String())
	if out != "" {
		logs.Debug("Detected username via git config user.name: %s", out)
	}
	return out
}
