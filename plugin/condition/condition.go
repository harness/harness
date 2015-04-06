package condition

import (
	"path/filepath"
	"strings"
)

type Condition struct {
	Owner       string // Indicates the step should run only for this repo (useful for forks)
	Branch      string // Indicates the step should run only for this branch
	Condition   string // Indicates the step should run if bash condition evals to true
	PullRequest *bool  `yaml:"pull_requests"` // Indicates the step should run for all pull requests
	AllBranches *bool  `yaml:"all_branches"`  // Indicates the step should run for all branches

	// Indicates the step should only run when the following
	// matrix values are present for the sub-build.
	Matrix map[string]string
}

// MatchPullRequest is a helper function that returns false
// if Pull Requests are disbled, but the pull request string
// is not empty.
func (c *Condition) MatchPullRequest(pr string) bool {
	if len(pr) == 0 {
		return true
	}
	if c.PullRequest == nil {
		return false
	}
	return *c.PullRequest
}

// MatchBranch is a helper function that returns true
// if all_branches is true. Else it returns false if a
// branch condition is specified, and the branch does
// not match.
func (c *Condition) MatchBranch(branch string) bool {
	if len(c.Branch) == 0 {
		return true
	}
	if c.AllBranches != nil && *c.AllBranches == true {
		return true
	}
	match, _ := filepath.Match(c.Branch, branch)
	return match
}

// MatchOwner is a helper function that returns false
// if an owner condition is specified and the repository
// owner does not match.
//
// This is useful when you want to prevent forks from
// executing deployment, publish or notification steps.
func (c *Condition) MatchOwner(owner string) bool {
	if len(c.Owner) == 0 {
		return true
	}
	parts := strings.Split(owner, "/")
	switch len(parts) {
	case 2:
		return c.Owner == parts[0]
	case 3:
		return c.Owner == parts[1]
	default:
		return c.Owner == owner
	}
}
