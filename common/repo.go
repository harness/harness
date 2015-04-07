package common

type Repo struct {
	ID       int64  `json:"id"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Language string `json:"language"`
	Private  bool   `json:"private"`
	Link     string `json:"link_url"`
	Clone    string `json:"clone_url"`
	Branch   string `json:"default_branch"`

	Timeout    int64 `json:"timeout"`
	Trusted    bool  `json:"trusted"`
	Disabled   bool  `json:"disabled"`
	DisablePR  bool  `json:"disable_prs"`
	DisableTag bool  `json:"disable_tags"`

	Created int64 `json:"created_at"`
	Updated int64 `json:"updated_at"`

	User *User  `json:"user,omitempty"`
	Last *Build `json:"last_build,omitempty"`
}

// Keypair represents an RSA public and private key
// assigned to a repository. It may be used to clone
// private repositories, or as a deployment key.
type Keypair struct {
	Public  string `json:"public"`
	Private string `json:"-"`
}

// Subscriber represents a user's subscription
// to a repository. This determines if the repository
// is displayed on the user dashboard and in the user
// event feed.
type Subscriber struct {
	Login string `json:"login,omitempty"`

	// Determines if notifications should be
	// received from this repository.
	Subscribed bool `json:"subscribed"`

	// Determines if all notifications should be
	// blocked from this repository.
	Ignored bool `json:"ignored"`
}

// Perm represents a user's permissiont to access
// a repository. Pull indicates read-only access. Push
// indiates write access. Admin indicates god access.
type Perm struct {
	Login string `json:"login,omitempty"`
	Pull  bool   `json:"pull"`
	Push  bool   `json:"push"`
	Admin bool   `json:"admin"`
}
