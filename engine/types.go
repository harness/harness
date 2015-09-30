package engine

import (
	"github.com/drone/drone/model"
)

type Event struct {
	Name string
	Msg  []byte
}

type Task struct {
	User      *model.User   `json:"-"`
	Repo      *model.Repo   `json:"repo"`
	Build     *model.Build  `json:"build"`
	BuildPrev *model.Build  `json:"build_last"`
	Jobs      []*model.Job  `json:"-"`
	Job       *model.Job    `json:"job"`
	Keys      *model.Key    `json:"keys"`
	Netrc     *model.Netrc  `json:"netrc"`
	Config    string        `json:"config"`
	Secret    string        `json:"secret"`
	System    *model.System `json:"system"`
}
