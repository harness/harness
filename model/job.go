package model

// swagger:model job
type Job struct {
	ID       int64  `json:"id"           meddler:"job_id,pk"`
	BuildID  int64  `json:"-"            meddler:"job_build_id"`
	NodeID   int64  `json:"-"            meddler:"job_node_id"`
	Number   int    `json:"number"       meddler:"job_number"`
	Status   string `json:"status"       meddler:"job_status"`
	ExitCode int    `json:"exit_code"    meddler:"job_exit_code"`
	Enqueued int64  `json:"enqueued_at"  meddler:"job_enqueued"`
	Started  int64  `json:"started_at"   meddler:"job_started"`
	Finished int64  `json:"finished_at"  meddler:"job_finished"`

	Environment map[string]string `json:"environment" meddler:"job_environment,json"`
}
