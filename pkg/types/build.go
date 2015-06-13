package types

type Build struct {
	ID       int64  `meddler:"build_id,pk"     json:"id"`
	CommitID int64  `meddler:"build_commit_id" json:"-"             sql:"unique:ux_build_seq,index:ix_build_commit_id"`
	State    string `meddler:"build_state"     json:"state"`
	ExitCode int    `meddler:"build_exit_code" json:"exit_code"`
	Sequence int    `meddler:"build_sequence"  json:"sequence"      sql:"unique:ux_build_seq"`
	Duration int64  `meddler:"build_duration"  json:"duration"`
	Started  int64  `meddler:"build_started"   json:"started_at"`
	Finished int64  `meddler:"build_finished"  json:"finished_at"`
	Created  int64  `meddler:"build_created"   json:"created_at"`
	Updated  int64  `meddler:"build_updated"   json:"updated_at"`

	Environment map[string]string `meddler:"build_environment,json" json:"environment" sql:"type:varchar,size:2048"`
}

// QUESTION: should we track if it was oom killed?
// OOMKill bool `meddler:"build_oom" json:"oom_kill"`
