package model

import "strconv"

type RepoLite struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar_url"`
}

// Repo represents a repository.
//
// swagger:model repo
type Repo struct {
	ID          int64  `json:"id"                meddler:"repo_id,pk"`
	UserID      int64  `json:"-"                 meddler:"repo_user_id"`
	Owner       string `json:"owner"             meddler:"repo_owner"`
	Name        string `json:"name"              meddler:"repo_name"`
	FullName    string `json:"full_name"         meddler:"repo_full_name"`
	Avatar      string `json:"avatar_url"        meddler:"repo_avatar"`
	Link        string `json:"link_url"          meddler:"repo_link"`
	Kind        string `json:"scm"               meddler:"repo_scm"`
	Clone       string `json:"clone_url"         meddler:"repo_clone"`
	Branch      string `json:"default_branch"    meddler:"repo_branch"`
	Timeout     int64  `json:"timeout"           meddler:"repo_timeout"`
	IsPrivate   bool   `json:"private"           meddler:"repo_private"`
	IsTrusted   bool   `json:"trusted"           meddler:"repo_trusted"`
	IsStarred   bool   `json:"starred,omitempty" meddler:"-"`
	AllowPull   bool   `json:"allow_pr"          meddler:"repo_allow_pr"`
	AllowPush   bool   `json:"allow_push"        meddler:"repo_allow_push"`
	AllowDeploy bool   `json:"allow_deploys"     meddler:"repo_allow_deploys"`
	AllowTag    bool   `json:"allow_tags"        meddler:"repo_allow_tags"`
	Hash        string `json:"-"                 meddler:"repo_hash"`
}

// ToEnv returns environment variable valus for the repository.
func (r *Repo) ToEnv(to map[string]string) {
	to["CI_VCS"] = r.Kind
	to["CI_REPO"] = r.FullName
	to["CI_REPO_OWNER"] = r.Owner
	to["CI_REPO_NAME"] = r.Name
	to["CI_REPO_LINK"] = r.Link
	to["CI_REPO_AVATAR"] = r.Avatar
	to["CI_REPO_BRANCH"] = r.Branch
	to["CI_REPO_PRIVATE"] = strconv.FormatBool(r.IsPrivate)
	to["CI_REPO_TRUSTED"] = strconv.FormatBool(r.IsTrusted)
	to["CI_REMOTE_URL"] = r.Clone
}
