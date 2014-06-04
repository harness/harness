package build

const (
	StatusNone    = "None"
	StatusEnqueue = "Pending"
	StatusStarted = "Started"
	StatusSuccess = "Success"
	StatusFailure = "Failure"
	StatusError   = "Error"
)

type Build struct {
	ID       int64  `meddler:"build_id,pk"     json:"id"`
	CommitID int64  `meddler:"commit_id"       json:"-"`
	Number   int64  `meddler:"build_number"    json:"number"`
	Matrix   string `meddler:"build_matrix"    json:"matrix"`
	Status   string `meddler:"build_status"    json:"status"`
	Started  int64  `meddler:"build_started"   json:"started_at"`
	Finished int64  `meddler:"build_finished"  json:"finished_at"`
	Duration int64  `meddler:"build_duration"  json:"duration"`
	Created  int64  `meddler:"build_created"   json:"created_at"`
	Updated  int64  `meddler:"build_updated"   json:"updated_at"`
}

// IsRunning returns true if the Build statis is Started
// or Pending, indicating it is currently running.
func (b *Build) IsRunning() bool {
	return (b.Status == StatusStarted || b.Status == StatusEnqueue)
}
