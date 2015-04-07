package common

type Hook struct {
	Repo        *Repo
	Commit      *Commit
	PullRequest *PullRequest
}
