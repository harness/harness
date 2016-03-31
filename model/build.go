package model

// swagger:model build
type Build struct {
	ID        int64  `json:"id"            meddler:"build_id,pk"`
	RepoID    int64  `json:"-"             meddler:"build_repo_id"`
	Number    int    `json:"number"        meddler:"build_number"`
	Event     string `json:"event"         meddler:"build_event"`
	Status    string `json:"status"        meddler:"build_status"`
	Enqueued  int64  `json:"enqueued_at"   meddler:"build_enqueued"`
	Created   int64  `json:"created_at"    meddler:"build_created"`
	Started   int64  `json:"started_at"    meddler:"build_started"`
	Finished  int64  `json:"finished_at"   meddler:"build_finished"`
	Deploy    string `json:"deploy_to"     meddler:"build_deploy"`
	Commit    string `json:"commit"        meddler:"build_commit"`
	Branch    string `json:"branch"        meddler:"build_branch"`
	Ref       string `json:"ref"           meddler:"build_ref"`
	Refspec   string `json:"refspec"       meddler:"build_refspec"`
	Remote    string `json:"remote"        meddler:"build_remote"`
	Title     string `json:"title"         meddler:"build_title"`
	Message   string `json:"message"       meddler:"build_message"`
	Timestamp int64  `json:"timestamp"     meddler:"build_timestamp"`
	Author    string `json:"author"        meddler:"build_author"`
	Avatar    string `json:"author_avatar" meddler:"build_avatar"`
	Email     string `json:"author_email"  meddler:"build_email"`
	Link      string `json:"link_url"      meddler:"build_link"`
}

type BuildGroup struct {
	Date   string
	Builds []*Build
}
