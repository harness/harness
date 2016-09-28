package model

// Work represents an item for work to be
// processed by a worker.
type Work struct {
	Signed    bool      `json:"signed"`
	Verified  bool      `json:"verified"`
	Yaml      string    `json:"config"`
	YamlEnc   string    `json:"secret"`
	Repo      *Repo     `json:"repo"`
	Build     *Build    `json:"build"`
	BuildLast *Build    `json:"build_last"`
	Job       *Job      `json:"job"`
	Netrc     *Netrc    `json:"netrc"`
	Keys      *Key      `json:"keys"`
	System    *System   `json:"system"`
	Secrets   []*Secret `json:"secrets"`
	User      *User     `json:"user"`
}
