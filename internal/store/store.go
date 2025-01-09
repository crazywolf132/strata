package store

import (
	"fmt"
	"os"
	"path/filepath"

	"strata/internal/git"
	"strata/internal/logs"
	"strata/internal/model"

	"gopkg.in/yaml.v3"
)

// We keep the stack data in .git/repo.strata for better Git integration
const StackFileName = "repo.strata"

// getStackPath returns the full path to the stack file in the .git directory
func getStackPath() (string, error) {
	gitDir, err := git.GetGitDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate .git directory: %v", err)
	}
	return filepath.Join(gitDir, StackFileName), nil
}

// LoadStack reads the stack data from disk
func LoadStack() (model.StackTree, error) {
	p, err := getStackPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		// If file doesn't exist, we can initialize an empty stack
		logs.Info("No existing stack file found. Creating new empty stack.")
		return model.StackTree{}, nil
	}
	content, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack file: %v", err)
	}
	var st model.StackTree
	if err := yaml.Unmarshal(content, &st); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stack file: %v", err)
	}
	return st, nil
}

// SaveStack writes the stack data to disk
func SaveStack(st model.StackTree) error {
	out, err := yaml.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal stack data: %v", err)
	}
	p, err := getStackPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, out, 0644); err != nil {
		return fmt.Errorf("failed to write stack file: %v", err)
	}
	return nil
}
