package store

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"strata/internal/config"
	"strata/internal/logs"
	"strata/internal/model"
)

// We keep the stack data in .strata_repo_stack.yaml for clarity, separate from config.
const StackFileName = "strata_repo_stack.yaml"

// LoadStack reads the stack data from disk
func LoadStack() (model.StackTree, error) {
	p := filepath.Join(".", StackFileName)
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
	p := filepath.Join(".", StackFileName)
	if err := os.WriteFile(p, out, 0644); err != nil {
		return fmt.Errorf("failed to write stack file: %v", err)
	}
	return nil
}

// The local config could define a custom path if needed, e.g., "stack_file = custom_stack.yml"
func getStackPath() string {
	custom := config.GetConfigValue("stack_file")
	if custom != "" {
		return custom
	}
	return StackFileName
}
