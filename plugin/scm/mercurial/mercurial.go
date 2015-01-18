package mercurial

import (
	"fmt"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

type Mercurial struct{}

func New() *Mercurial {
	return &Mercurial{}
}

func (m *Mercurial) Commit(f *buildfile.Buildfile, r *repo.Repo) {
	f.WriteCmd(fmt.Sprintf("hg clone --branch %s %s %s", m.GetBranch(r), r.Path, r.Dir))
	f.WriteCmd(fmt.Sprintf("hg update %s", r.Commit))
}

func (m *Mercurial) PullRequest(f *buildfile.Buildfile, r *repo.Repo) {
	return
}

func (m *Mercurial) DefaultBranch() string {
	return "default"
}

func (m *Mercurial) GetBranch(r *repo.Repo) string {
	if r.Branch == "" {
		return m.DefaultBranch()
	} else {
		return r.Branch
	}
}

func (m *Mercurial) GetKind() string {
	return "mercurial"
}
