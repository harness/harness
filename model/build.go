package model

// swagger:model build
type Build struct {
	ID        int64   `json:"id"            meddler:"build_id,pk"`
	RepoID    int64   `json:"-"             meddler:"build_repo_id"`
	ConfigID  int64   `json:"-"             meddler:"build_config_id"`
	Number    int     `json:"number"        meddler:"build_number"`
	Parent    int     `json:"parent"        meddler:"build_parent"`
	Event     string  `json:"event"         meddler:"build_event"`
	Status    string  `json:"status"        meddler:"build_status"`
	Error     string  `json:"error"         meddler:"build_error"`
	Enqueued  int64   `json:"enqueued_at"   meddler:"build_enqueued"`
	Created   int64   `json:"created_at"    meddler:"build_created"`
	Started   int64   `json:"started_at"    meddler:"build_started"`
	Finished  int64   `json:"finished_at"   meddler:"build_finished"`
	Deploy    string  `json:"deploy_to"     meddler:"build_deploy"`
	Commit    string  `json:"commit"        meddler:"build_commit"`
	Branch    string  `json:"branch"        meddler:"build_branch"`
	Ref       string  `json:"ref"           meddler:"build_ref"`
	Refspec   string  `json:"refspec"       meddler:"build_refspec"`
	Remote    string  `json:"remote"        meddler:"build_remote"`
	Title     string  `json:"title"         meddler:"build_title"`
	Message   string  `json:"message"       meddler:"build_message"`
	Timestamp int64   `json:"timestamp"     meddler:"build_timestamp"`
	Sender    string  `json:"sender"        meddler:"build_sender"`
	Author    string  `json:"author"        meddler:"build_author"`
	Avatar    string  `json:"author_avatar" meddler:"build_avatar"`
	Email     string  `json:"author_email"  meddler:"build_email"`
	Link      string  `json:"link_url"      meddler:"build_link"`
	Signed    bool    `json:"signed"        meddler:"build_signed"`   // deprecate
	Verified  bool    `json:"verified"      meddler:"build_verified"` // deprecate
	Reviewer  string  `json:"reviewed_by"   meddler:"build_reviewer"`
	Reviewed  int64   `json:"reviewed_at"   meddler:"build_reviewed"`
	Procs     []*Proc `json:"procs,omitempty" meddler:"-"`
}

// Trim trims string values that would otherwise exceed
// the database column sizes and fail to insert.
func (b *Build) Trim() {
	if len(b.Title) > 1000 {
		b.Title = b.Title[:1000]
	}
	if len(b.Message) > 2000 {
		b.Message = b.Message[:2000]
	}
}
