package queue

import "github.com/drone/drone/model"

// Work represents an item for work to be
// processed by a worker.
type Work struct {
	Signed    bool            `json:"signed"`
	Verified  bool            `json:"verified"`
	Yaml      string          `json:"config"`
	YamlEnc   string          `json:"secret"`
	Repo      *model.Repo     `json:"repo"`
	Build     *model.Build    `json:"build"`
	BuildLast *model.Build    `json:"build_last"`
	Job       *model.Job      `json:"job"`
	Netrc     *model.Netrc    `json:"netrc"`
	Keys      *model.Key      `json:"keys"`
	System    *model.System   `json:"system"`
	Secrets   []*model.Secret `json:"secrets"`
	User      *model.User     `json:"user"`
}
