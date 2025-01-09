package model

import "time"

type StackNode struct {
	BranchName   string   `yaml:"branch_name"`
	ParentBranch string   `yaml:"parent_branch,omitempty"`
	Children     []string `yaml:"children,omitempty"`

	CreatedBy string    `yaml:"created_by,omitempty"` // GH username or fallback
	CreatedAt time.Time `yaml:"created_at,omitempty"`
	UpdatedAt time.Time `yaml:"updated_at,omitempty"`
}

type StackTree map[string]*StackNode
