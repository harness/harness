package common

const (
	StatePending = "pending"
	StateRunning = "running"
	StateSuccess = "success"
	StateFailure = "failure"
	StateKilled  = "killed"
	StateError   = "error"
)

type Build struct {
	Number   int    `json:"number"`
	State    string `json:"state"`
	Duration int64  `json:"duration"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`
	Created  int64  `json:"created_at"`
	Updated  int64  `json:"updated_at"`

	// Tasks int  `json:"task_count"`

	// Commit represents the commit data send in the
	// post-commit hook. This will not be populated when
	// a pull requests.
	Commit *Commit `json:"head_commit,omitempty"`

	// PullRequest represents the pull request data sent
	// in the post-commit hook. This will only be populated
	// when a pull request.
	PullRequest *PullRequest `json:"pull_request,omitempty"`

	// Statuses represents a list of build statuses used
	// to annotate the build.
	Statuses []*Status `json:"statuses,omitempty"`

	// Tasks represents a list of build tasks. A build is
	// comprised of one or many tasks.
	Tasks []*Task `json:"tasks,omitempty"`
}

type Status struct {
	State   string `json:"state"`
	Link    string `json:"target_url"`
	Desc    string `json:"description"`
	Context string `json:"context"`
}

type Commit struct {
	Sha       string  `json:"sha,omitempty"`
	Ref       string  `json:"ref,omitempty"`
	Message   string  `json:"message,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
	Author    *Author `json:"author,omitempty"`
	Remote    *Remote `json:"repo,omitempty"`
}

type PullRequest struct {
	Number int     `json:"number,omitempty"`
	Title  string  `json:"title,omitempty"`
	Source *Commit `json:"source,omitempty"`
	Target *Commit `json:"target,omitempty"`
}

type Author struct {
	Name     string `json:"name,omitempty"`
	Login    string `json:"login,omitempty"`
	Email    string `json:"email,omitempty"`
	Gravatar string `json:"gravatar_id,omitempty"`
}

type Remote struct {
	Name     string `json:"name,omitempty"`
	FullName string `json:"full_name,omitempty"`
	Clone    string `json:"clone_url,omitempty"`
}
