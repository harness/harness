package model

// Hook represents a subset of commit meta-data provided
// by post-commit and pull request hooks.
type Hook struct {
	Owner       string
	Repo        string
	Sha         string
	Branch      string
	PullRequest string
	Author      string
	Gravatar    string
	Timestamp   string
	Message     string
}
