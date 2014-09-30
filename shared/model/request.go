package model

type Request struct {
	Host   string  `json:"-"`
	User   *User   `json:"-"`
	Repo   *Repo   `json:"repo"`
	Commit *Commit `json:"commit"`
}
