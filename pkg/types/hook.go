package types

type Hook struct {
	Repo        *Repo
	Commit      *Commit
	PullRequest *PullRequest
}
