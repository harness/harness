package model

// Feed represents an item in the user's feed or timeline.
//
// swagger:model feed
type Feed struct {
	Owner    string `json:"owner"         meddler:"repo_owner"`
	Name     string `json:"name"          meddler:"repo_name"`
	FullName string `json:"full_name"     meddler:"repo_full_name"`

	Number   int    `json:"number,omitempty"        meddler:"build_number,zeroisnull"`
	Event    string `json:"event,omitempty"         meddler:"build_event,zeroisnull"`
	Status   string `json:"status,omitempty"        meddler:"build_status,zeroisnull"`
	Created  int64  `json:"created_at,omitempty"    meddler:"build_created,zeroisnull"`
	Started  int64  `json:"started_at,omitempty"    meddler:"build_started,zeroisnull"`
	Finished int64  `json:"finished_at,omitempty"   meddler:"build_finished,zeroisnull"`
	Commit   string `json:"commit,omitempty"        meddler:"build_commit,zeroisnull"`
	Branch   string `json:"branch,omitempty"        meddler:"build_branch,zeroisnull"`
	Ref      string `json:"ref,omitempty"           meddler:"build_ref,zeroisnull"`
	Refspec  string `json:"refspec,omitempty"       meddler:"build_refspec,zeroisnull"`
	Remote   string `json:"remote,omitempty"        meddler:"build_remote,zeroisnull"`
	Title    string `json:"title,omitempty"         meddler:"build_title,zeroisnull"`
	Message  string `json:"message,omitempty"       meddler:"build_message,zeroisnull"`
	Author   string `json:"author,omitempty"        meddler:"build_author,zeroisnull"`
	Avatar   string `json:"author_avatar,omitempty" meddler:"build_avatar,zeroisnull"`
	Email    string `json:"author_email,omitempty"  meddler:"build_email,zeroisnull"`
}
