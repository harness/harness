package model

type Key struct {
	ID      int64  `json:"-"       meddler:"key_id,pk"`
	RepoID  int64  `json:"-"       meddler:"key_repo_id"`
	Public  string `json:"public"  meddler:"key_public"`
	Private string `json:"private" meddler:"key_private"`
}
