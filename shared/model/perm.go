package model

type Perm struct {
	ID      int64 `meddler:"perm_id,pk"   json:"-"`
	UserID  int64 `meddler:"user_id"      json:"-"`
	RepoID  int64 `meddler:"repo_id"      json:"-"`
	Read    bool  `meddler:"perm_read"    json:"read"`
	Write   bool  `meddler:"perm_write"   json:"write"`
	Admin   bool  `meddler:"perm_admin"   json:"admin"`
	Guest   bool  `meddler:"-"            json:"guest"`
	Created int64 `meddler:"perm_created" json:"-"`
	Updated int64 `meddler:"perm_updated" json:"-"`
}
