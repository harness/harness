package types

type Status struct {
	ID       int64  `meddler:"status_id,pk"    json:"-"`
	CommitID int64  `meddler:"commit_id"       json:"-"`
	State    string `meddler:"status_state"    json:"state"`
	Link     string `meddler:"status_link"     json:"target_url"`
	Desc     string `meddler:"status_desc"     json:"description"`
	Context  string `meddler:"status_context"  json:"context"`
}
