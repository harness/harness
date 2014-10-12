package git

import (
	"fmt"
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

const (
	// Gommand to the current commit hash
	CmdRevParse = "COMMIT=$(git rev-parse HEAD)"

	// Command to set the git user and email based on the
	// individual that made the commit.
	CmdGlobalEmail = "git config --global user.email $(git --no-pager log -1 --pretty=format:'%ae')"
	CmdGlobalUser  = "git config --global user.name  $(git --no-pager log -1 --pretty=format:'%an')"
)

type Git struct {
	Target string `yaml:"target,omitempty"`
	Force  bool   `yaml:"force,omitempty"`
	Branch string `yaml:"branch,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (g *Git) Write(f *buildfile.Buildfile) {
	f.WriteCmdSilent(CmdRevParse)
	f.WriteCmdSilent(CmdGlobalUser)
	f.WriteCmdSilent(CmdGlobalEmail)

	// add target as a git remote
	f.WriteCmd(fmt.Sprintf("git remote add deploy %s", g.Target))

	dest := g.Branch
	if len(dest) == 0 {
		dest = "master"
	}

	switch g.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to git remote.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'add build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push deploy HEAD:%s --force", dest))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push deploy $COMMIT:%s", dest))
	}
}

func (g *Git) GetCondition() *condition.Condition {
	return g.Condition
}
