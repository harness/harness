package perm

type perm struct {
	ID      int64 `meddler:"perm_id,pk"`
	UserID  int64 `meddler:"user_id"`
	RepoID  int64 `meddler:"repo_id"`
	Read    bool  `meddler:"perm_read"`
	Write   bool  `meddler:"perm_write"`
	Admin   bool  `meddler:"perm_admin"`
	Created int64 `meddler:"perm_created"`
	Updated int64 `meddler:"perm_updated"`
}
