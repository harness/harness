package types

const (
	StatePending = "pending"
	StateRunning = "running"
	StateSuccess = "success"
	StateFailure = "failure"
	StateKilled  = "killed"
	StateError   = "error"
)

type Build struct {
	ID       int64
	RepoID   int64  `json:"-"      sql:"unique:ux_build_number,index:ix_build_repo_id"`
	Number   int    `json:"number" sql:"unique:ux_build_number"`
	Status   string `json:"status"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`

	Commit      *Commit      `json:"head_commit"`
	PullRequest *PullRequest `json:"pull_request"`

	Jobs []*Job `json:"jobs,omitempty" sql:"-"`
}

type PullRequest struct {
	Number int     `json:"number"`
	Title  string  `json:"title"`
	Base   *Commit `json:"base_commit"`
}

type Commit struct {
	Sha       string  `json:"sha"`
	Ref       string  `json:"ref"`
	Branch    string  `json:"branch" sql:"index:ix_commit_branch"`
	Message   string  `json:"message"`
	Timestamp string  `json:"timestamp"`
	Remote    string  `json:"remote"`
	Author    *Author `json:"author"`
}

type Author struct {
	Login string `json:"login"`
	Email string `json:"email"`
}
