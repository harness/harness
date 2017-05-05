package model

type SenderService interface {
	SenderAllowed(*User, *Repo, *Build, *Config) (bool, error)
	SenderCreate(*Repo, *Sender) error
	SenderUpdate(*Repo, *Sender) error
	SenderDelete(*Repo, string) error
	SenderList(*Repo) ([]*Sender, error)
}

type SenderStore interface {
	SenderFind(*Repo, string) (*Sender, error)
	SenderList(*Repo) ([]*Sender, error)
	SenderCreate(*Sender) error
	SenderUpdate(*Sender) error
	SenderDelete(*Sender) error
}

type Sender struct {
	ID     int64    `json:"-"      meddler:"sender_id,pk"`
	RepoID int64    `json:"-"      meddler:"sender_repo_id"`
	Login  string   `json:"login"  meddler:"sender_login"`
	Allow  bool     `json:"allow"  meddler:"sender_allow"`
	Block  bool     `json:"block"  meddler:"sender_block"`
	Branch []string `json:"branch" meddler:"-"`
	Deploy []string `json:"deploy" meddler:"-"`
	Event  []string `json:"event"  meddler:"-"`
}
