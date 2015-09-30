package model

const (
	EventPush   = "push"
	EventPull   = "pull_request"
	EventTag    = "tag"
	EventDeploy = "deploy"
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
