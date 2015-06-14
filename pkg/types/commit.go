package types

const (
	StatePending = "pending"
	StateRunning = "running"
	StateSuccess = "success"
	StateFailure = "failure"
	StateKilled  = "killed"
	StateError   = "error"
)

type Commit struct {
	ID           int64  `meddler:"commit_id,pk"         json:"id"`
	RepoID       int64  `meddler:"commit_repo_id"       json:"-"              sql:"unique:ux_commit_seq,index:ix_commit_repo_id"`
	Sequence     int    `meddler:"commit_sequence"      json:"sequence"       sql:"unique:ux_commit_seq"`
	State        string `meddler:"commit_state"         json:"state"`
	Started      int64  `meddler:"commit_started"       json:"started_at"`
	Finished     int64  `meddler:"commit_finished"      json:"finished_at"`
	Sha          string `meddler:"commit_sha"           json:"sha"`
	Ref          string `meddler:"commit_ref"           json:"ref"`
	PullRequest  string `meddler:"commit_pull_request"  json:"pull_request,omitempty"`
	Branch       string `meddler:"commit_branch"        json:"branch"`
	Author       string `meddler:"commit_author"        json:"author"`
	Gravatar     string `meddler:"commit_gravatar"      json:"gravatar"`
	Timestamp    string `meddler:"commit_timestamp"     json:"timestamp"`
	Message      string `meddler:"commit_message"       json:"message"`
	SourceRemote string `meddler:"commit_source_remote" json:"source_remote,omitempty"`
	SourceBranch string `meddler:"commit_source_branch" json:"source_branch,omitempty"`
	SourceSha    string `meddler:"commit_source_sha"    json:"source_sha,omitempty"`
	Created      int64  `meddler:"commit_created"       json:"created_at"`
	Updated      int64  `meddler:"commit_updated"       json:"updated_at"`

	Builds []*Build `meddler:"-" json:"builds,omitempty" sql:"-"`
}
