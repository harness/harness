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
	RepoID   int64  `json:"id"     sql:"unique:ux_build_number,index:ix_build_repo_id"`
	Number   int    `json:"number" sql:"unique:ux_build_number"`
	Event    string `json:"event"`
	Status   string `json:"status"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`

	Commit      *Commit      `json:"head_commit"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`

	Jobs []*Job `json:"jobs,omitempty" sql:"-"`
}

type PullRequest struct {
	Number int     `json:"number,omitempty"`
	Title  string  `json:"title,omitempty"`
	Link   string  `json:"link_url,omitempty"`
	Base   *Commit `json:"base_commit,omitempty"`
}

type Commit struct {
	Sha       string  `json:"sha"`
	Ref       string  `json:"ref"`
	Link      string  `json:"link_url,omitempty"`
	Branch    string  `json:"branch" sql:"index:ix_commit_branch"`
	Message   string  `json:"message"`
	Timestamp string  `json:"timestamp,omitempty"`
	Remote    string  `json:"remote,omitempty"`
	Author    *Author `json:"author,omitempty"`
}

type Author struct {
	Login string `json:"login,omitempty"`
	Email string `json:"email,omitempty"`
}
