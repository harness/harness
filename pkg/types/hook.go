package types

type Hook struct {
	Event       string
	Repo        *Repo
	Commit      *Commit
	PullRequest *PullRequest
}
