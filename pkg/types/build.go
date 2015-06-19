package types

type Job struct {
	ID       int64  `json:"id"`
	BuildID  int64  `json:"-"         sql:"unique:ux_build_number,index:ix_job_build_id"`
	Number   int    `json:"number"    sql:"unique:ux_build_number"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`

	Environment map[string]string `json:"environment" sql:"type:varchar,size:2048"`
}
