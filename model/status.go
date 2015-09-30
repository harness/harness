package types

type Status struct {
	ID       int64  `json:"-"`
	CommitID int64  `json:"-"`
	State    string `json:"status"`
	Link     string `json:"target_url"`
	Desc     string `json:"description"`
	Context  string `json:"context"`
}
