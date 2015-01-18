package scm

import (
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

type Scm interface {
	Commit(f *buildfile.Buildfile, r *repo.Repo)
	PullRequest(f *buildfile.Buildfile, r *repo.Repo)
	DefaultBranch() string
	GetBranch(r *repo.Repo) string
	GetKind() string
}

var scms []Scm

func Register(scm Scm) {
	scms = append(scms, scm)
}

func Lookup(name string) Scm {
	for _, scm := range scms {
		if scm.GetKind() == name {
			return scm
		}
	}
	return nil
}
