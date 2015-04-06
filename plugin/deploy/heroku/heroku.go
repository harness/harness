package heroku

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

	// Command to write the API token to ~/.netrc
	// use "_" since heroku git authentication ignores username
	CmdLogin = "echo 'machine git.heroku.com login _ password %s' >> ~/.netrc"
)

type Heroku struct {
	App   string `yaml:"app,omitempty"`
	Force bool   `yaml:"force,omitempty"`
	Token string `yaml:"token,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (h *Heroku) Write(f *buildfile.Buildfile) {
	f.WriteCmdSilent(CmdRevParse)
	f.WriteCmdSilent(CmdGlobalUser)
	f.WriteCmdSilent(CmdGlobalEmail)
	f.WriteCmdSilent(fmt.Sprintf(CmdLogin, h.Token))

	// add heroku as a git remote
	f.WriteCmd(fmt.Sprintf("git remote add heroku https://git.heroku.com/%s.git", h.App))

	switch h.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to Heroku.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'adding build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push heroku HEAD:refs/heads/master --force"))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push heroku $COMMIT:refs/heads/master"))
	}
}

func (h *Heroku) GetCondition() *condition.Condition {
	return h.Condition
}
