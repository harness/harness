package model

type Registry struct {
	ID       int64  `json:"id"       meddler:"registry_id,pk"`
	RepoID   int64  `json:"-"        meddler:"registry_repo_id"`
	Addr     string `json:"addr"     meddler:"registry_addr"`
	Username string `json:"username" meddler:"registry_username"`
	Password string `json:"password" meddler:"registry_password"`
	Email    string `json:"email"    meddler:"registry_email"`
	Token    string `json:"token"    meddler:"registry_token"`
}

func (r *Registry) Validate() error {
	return nil
}
