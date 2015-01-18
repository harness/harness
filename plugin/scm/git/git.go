package git

import (
	"fmt"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

type Git struct{}

func New() *Git {
	return &Git{}
}

func (g *Git) Commit(f *buildfile.Buildfile, r *repo.Repo) {
	f.WriteCmd(fmt.Sprintf("git clone --depth=%d --recursive %s %s", r.Depth, r.Path, r.Dir))
	f.WriteCmd(fmt.Sprintf("git checkout -qf %s", r.Commit))
}

func (g *Git) PullRequest(f *buildfile.Buildfile, r *repo.Repo) {
	f.WriteCmd(fmt.Sprintf("git clone --depth=%d --recursive %s %s", r.Depth, r.Path, r.Dir))
	f.WriteCmd(fmt.Sprintf("git fetch origin +refs/pull/%s/head:refs/remotes/origin/pr/%s", r.PR, r.PR))
	f.WriteCmd(fmt.Sprintf("git checkout -qf -b pr/%s origin/pr/%s", r.PR, r.PR))
}

func (g *Git) DefaultBranch() string {
	return "master"
}

func (g *Git) GetBranch(r *repo.Repo) string {
	if r.Branch == "" {
		return g.DefaultBranch()
	} else {
		return r.Branch
	}
}

func (g *Git) GetKind() string {
	return "git"
}
