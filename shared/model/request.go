package model

import (
	"fmt"
)

type Request struct {
	Host   string  `json:"-"`
	User   *User   `json:"-"`
	Repo   *Repo   `json:"repo"`
	Commit *Commit `json:"commit"`
	Prior  *Commit `json:"prior_commit"`
}

// URL returns the link to the commit in
// string format.
func (r *Request) URL() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s",
		r.Host,
		r.Repo.Host,
		r.Repo.Owner,
		r.Repo.Name,
		r.Commit.Branch,
		r.Commit.Sha,
	)
}
