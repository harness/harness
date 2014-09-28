package model

type Build struct {
	ID        int64  `meddler:"build_id,pk"      json:"id"`
	Index     int64  `meddler:"build_index"      json:"index"`
	Name      string `meddler:"build_name"       json:"name"`
	Status    string `meddler:"build_status"     json:"status"`
	AllowFail bool   `meddler:"build_allow_fail" json:"allow_fail"`
	Output    string `meddler:"build_output"     json:"output"`
	CommitID  int64  `meddler:"commit_id"        json:"commit_id"`
	Duration  int64  `meddler:"build_duration"   json:"duration"`
	Started   int64  `meddler:"build_started"    json:"started"`
	Finished  int64  `meddler:"build_finished"   json:"finished_at"`
	Created   int64  `meddler:"build_created"    json:"created_at"`
	Updated   int64  `meddler:"build_updated"    json:"updated_at"`
}
