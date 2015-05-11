package common

type Build struct {
	ID       int64  `meddler:"build_id,pk"    json:"-"`
	CommitID int64  `meddler:"commit_id"      json:"-"`
	State    string `meddler:"build_state"    json:"state"`
	ExitCode int    `meddler:"build_exit"     json:"exit_code"`
	Sequence int    `meddler:"build_seq"      json:"sequence"`
	Duration int64  `meddler:"build_duration" json:"duration"`
	Started  int64  `meddler:"build_started"  json:"started_at"`
	Finished int64  `meddler:"build_finished" json:"finished_at"`
	Created  int64  `meddler:"build_created"  json:"created_at"`
	Updated  int64  `meddler:"build_updated"  json:"updated_at"`

	Environment map[string]string `meddler:"build_env,json" json:"environment"`
}

// QUESTION: should we track if it was oom killed?
// OOMKill bool `meddler:"build_oom" json:"oom_kill"`
