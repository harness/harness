package model

// Feed represents an item in the user's feed or timeline.
//
// swagger:model feed
type Feed struct {
	Owner    string `json:"owner"         meddler:"repo_owner"`
	Name     string `json:"name"          meddler:"repo_name"`
	FullName string `json:"full_name"     meddler:"repo_full_name"`

	Number   int    `json:"number"        meddler:"build_number"`
	Event    string `json:"event"         meddler:"build_event"`
	Status   string `json:"status"        meddler:"build_status"`
	Created  int64  `json:"created_at"    meddler:"build_created"`
	Started  int64  `json:"started_at"    meddler:"build_started"`
	Finished int64  `json:"finished_at"   meddler:"build_finished"`
	Commit   string `json:"commit"        meddler:"build_commit"`
	Branch   string `json:"branch"        meddler:"build_branch"`
	Ref      string `json:"ref"           meddler:"build_ref"`
	Refspec  string `json:"refspec"       meddler:"build_refspec"`
	Remote   string `json:"remote"        meddler:"build_remote"`
	Title    string `json:"title"         meddler:"build_title"`
	Message  string `json:"message"       meddler:"build_message"`
	Author   string `json:"author"        meddler:"build_author"`
	Avatar   string `json:"author_avatar" meddler:"build_avatar"`
	Email    string `json:"author_email"  meddler:"build_email"`
}
