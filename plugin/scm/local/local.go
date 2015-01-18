// Abstract package over rsync, for local drone cli
package local

import (
	"fmt"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

type Local struct{}

func New() *Local {
	return &Local{}
}

func (l *Local) Commit(f *buildfile.Buildfile, r *repo.Repo) {
	f.WriteCmd(fmt.Sprintf("rsync -rt %s %s", r.Path, r.Dir))
}

func (l *Local) PullRequest(f *buildfile.Buildfile, r *repo.Repo) {
	return
}

func (l *Local) DefaultBranch() string {
	return ""
}

func (l *Local) GetBranch(r *repo.Repo) string {
	return l.DefaultBranch()
}

func (l *Local) GetKind() string {
	return "local"
}
