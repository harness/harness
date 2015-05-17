package types

type Repo struct {
	ID          int64  `meddler:"repo_id,pk"       json:"id"`
	UserID      int64  `meddler:"user_id"          json:"-"`
	Owner       string `meddler:"repo_owner"       json:"owner"`
	Name        string `meddler:"repo_name"        json:"name"`
	FullName    string `meddler:"repo_slug"        json:"full_name"`
	Token       string `meddler:"repo_token"       json:"-"`
	Language    string `meddler:"repo_lang"        json:"language"`
	Private     bool   `meddler:"repo_private"     json:"private"`
	Self        string `meddler:"repo_self"        json:"self_url"`
	Link        string `meddler:"repo_link"        json:"link_url"`
	Clone       string `meddler:"repo_clone"       json:"clone_url"`
	Branch      string `meddler:"repo_branch"      json:"default_branch"`
	Timeout     int64  `meddler:"repo_timeout"     json:"timeout"`
	Trusted     bool   `meddler:"repo_trusted"     json:"trusted"`
	PostCommit  bool   `meddler:"repo_push"        json:"post_commits"`
	PullRequest bool   `meddler:"repo_pull"        json:"pull_requests"`
	PublicKey   string `meddler:"repo_public_key"  json:"-"`
	PrivateKey  string `meddler:"repo_private_key" json:"-"`
	Created     int64  `meddler:"repo_created"     json:"created_at"`
	Updated     int64  `meddler:"repo_updated"     json:"updated_at"`

	Params map[string]string `meddler:"repo_params,json" json:"-"`
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
	FullName string `meddler:"repo_slug"       json:"full_name"`
	Number   int    `meddler:"commit_seq"      json:"number"`
	State    string `meddler:"commit_state"    json:"state"`
	Started  int64  `meddler:"commit_started"  json:"started_at"`
	Finished int64  `meddler:"commit_finished" json:"finished_at"`
}

type Perm struct {
	Pull  bool
	Push  bool
	Admin bool
}
