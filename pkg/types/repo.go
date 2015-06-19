package types

type Repo struct {
	ID       int64  `meddler:"repo_id,pk"        json:"id"`
	UserID   int64  `meddler:"repo_user_id"      json:"-"          sql:"index:ix_repo_user_id"`
	Owner    string `meddler:"repo_owner"        json:"owner"      sql:"unique:ux_repo_owner_name"`
	Name     string `meddler:"repo_name"         json:"name"       sql:"unique:ux_repo_owner_name"`
	FullName string `meddler:"repo_full_name"    json:"full_name"  sql:"unique:ux_repo_full_name"`
	Self     string `meddler:"repo_self"         json:"self_url"`
	Link     string `meddler:"repo_link"         json:"link_url"`
	Clone    string `meddler:"repo_clone"        json:"clone_url"`
	Branch   string `meddler:"repo_branch"       json:"default_branch"`
	Private  bool   `meddler:"repo_private"      json:"private"`
	Trusted  bool   `meddler:"repo_trusted"      json:"trusted"`
	Timeout  int64  `meddler:"repo_timeout"      json:"timeout"`

	Keys  *Keypair `json:"-"`
	Hooks *Hooks   `json:"hooks"`

	// Perms are the current user's permissions to push,
	// pull, and administer this repository. The permissions
	// are sourced from the version control system (ie GitHub)
	Perms *Perm `json:"perms,omitempty" sql:"-"`

	// Params are private environment parameters that are
	// considered secret and are therefore stored external
	// to the source code repository inside Drone.
	Params map[string]string `json:"-"`
}

type RepoLite struct {
	ID       int64  `meddler:"repo_id,pk"   json:"id"`
	UserID   int64  `meddler:"user_id"      json:"-"`
	Owner    string `meddler:"repo_owner"   json:"owner"`
	Name     string `meddler:"repo_name"    json:"name"`
	FullName string `meddler:"repo_slug"    json:"full_name"`
	Language string `meddler:"repo_lang"    json:"language"`
	Private  bool   `meddler:"repo_private" json:"private"`
	Created  int64  `meddler:"repo_created" json:"created_at"`
	Updated  int64  `meddler:"repo_updated" json:"updated_at"`
}

type RepoCommit struct {
	ID       int64  `meddler:"repo_id,pk"      json:"id"`
	Owner    string `meddler:"repo_owner"      json:"owner"`
	Name     string `meddler:"repo_name"       json:"name"`
	FullName string `meddler:"repo_full_name"  json:"full_name"`
	Number   int    `meddler:"commit_sequence" json:"number"`
	State    string `meddler:"commit_state"    json:"state"`
	Started  int64  `meddler:"commit_started"  json:"started_at"`
	Finished int64  `meddler:"commit_finished" json:"finished_at"`
}

type Perm struct {
	Pull  bool `json:"pull"  sql:"-"`
	Push  bool `json:"push"  sql:"-"`
	Admin bool `json:"admin" sql:"-"`
}

type Hooks struct {
	PullRequest bool `json:"pull_request"`
	Push        bool `json:"push"`
	Tags        bool `json:"tags"`
}

// Keypair represents an RSA public and private key
// assigned to a repository. It may be used to clone
// private repositories, or as a deployment key.
type Keypair struct {
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
}
