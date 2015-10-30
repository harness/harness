package model

type Log struct {
	ID    int64  `meddler:"log_id,pk"`
	JobID int64  `meddler:"log_job_id"`
	Data  []byte `meddler:"log_data"`
}
