package model

const (
	EventPush   = "push"
	EventPull   = "pull_request"
	EventTag    = "tag"
	EventDeploy = "deployment"
)

const (
	StatusSkipped = "skipped"
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusKilled  = "killed"
	StatusError   = "error"
)

const (
	RepoGit      = "git"
	RepoHg       = "hg"
	RepoFossil   = "fossil"
	RepoPerforce = "perforce"
)
