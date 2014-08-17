package model

type Perm struct {
	Id      int64 `gorm:"primary_key:yes"   json:"-"`
	UserId  int64 `json:"-"`
	RepoId  int64 `json:"-"`
	Read    bool  `json:"read"`
	Write   bool  `json:"write"`
	Admin   bool  `json:"admin"`
	Guest   bool  `json:"guest" sql:"-"`
	Created int64 `json:"-"`
	Updated int64 `json:"-"`
}
