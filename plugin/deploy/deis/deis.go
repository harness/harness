package deis

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

// deploy:
//   deis:
//     app: safe-island-6261
//     deisurl: deis.myurl.tdl:2222/

type Deis struct {
	App       string               `yaml:"app,omitempty"`
	Force     bool                 `yaml:"force,omitempty"`
	Deisurl   string               `yaml:"deisurl,omitempty"`
	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (h *Deis) Write(f *buildfile.Buildfile) {
	f.WriteCmdSilent(CmdRevParse)
	f.WriteCmdSilent(CmdGlobalUser)
	f.WriteCmdSilent(CmdGlobalEmail)

	// git@deis.yourdomain.com:2222/drone.git

	f.WriteCmd(fmt.Sprintf("git remote add deis ssh://git@%s%s.git", h.Deisurl, h.App))

	switch h.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to Deis.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'adding build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push deis HEAD:master --force"))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push deis $COMMIT:master"))
	}
}

func (h *Deis) GetCondition() *condition.Condition {
	return h.Condition
}
