package tsuru

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

type Tsuru struct {
	Force  bool   `yaml:"force,omitempty"`
	Remote string `yaml:"remote,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (t *Tsuru) Write(f *buildfile.Buildfile) {
	f.WriteCmdSilent(CmdRevParse)
	f.WriteCmdSilent(CmdGlobalUser)
	f.WriteCmdSilent(CmdGlobalEmail)

	// add tsuru as a git remote
	f.WriteCmd(fmt.Sprintf("git remote add tsuru %s", t.Remote))

	switch t.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to Tsuru.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'adding build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push tsuru HEAD:master --force"))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push tsuru $COMMIT:master"))
	}
}

func (t *Tsuru) GetCondition() *condition.Condition {
	return t.Condition
}
