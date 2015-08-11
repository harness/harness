package types

type Repo struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"-"          sql:"index:ix_repo_user_id"`
	Owner    string `json:"owner"      sql:"unique:ux_repo_owner_name"`
	Name     string `json:"name"       sql:"unique:ux_repo_owner_name"`
	FullName string `json:"full_name"  sql:"unique:ux_repo_full_name"`
	Avatar   string `json:"avatar"`
	Self     string `json:"self_url"`
	Link     string `json:"link_url"`
	Clone    string `json:"clone_url"`
	Branch   string `json:"default_branch"`
	Private  bool   `json:"private"`
	Trusted  bool   `json:"trusted"`
	Timeout  int64  `json:"timeout"`

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

	// randomly generated hash used to sign repository
	// tokens and encrypt and decrypt private variables.
	Hash string `json:"-"`
}

type RepoLite struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"-"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Language string `json:"language"`
	Private  bool   `json:"private"`
	Created  int64  `json:"created_at"`
	Updated  int64  `json:"updated_at"`
}

type RepoCommit struct {
	ID       int64  `json:"id"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Number   int    `json:"number"`
	State    string `json:"state"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`
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
