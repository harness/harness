package deploy

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

type Heroku struct {
	App    string `yaml:"app,omitempty"`
	Force  bool   `yaml:"force,omitempty"`
	Branch string `yaml:"branch,omitempty"`
}

func (h *Heroku) Write(f *buildfile.Buildfile) {
	// get the current commit hash
	f.WriteCmdSilent("COMMIT=$(git rev-parse HEAD)")

	// set the git user and email based on the individual
	// that made the commit.
	f.WriteCmdSilent("git config --global user.name $(git --no-pager log -1 --pretty=format:'%an')")
	f.WriteCmdSilent("git config --global user.email $(git --no-pager log -1 --pretty=format:'%ae')")

	// add heroku as a git remote
	f.WriteCmd(fmt.Sprintf("git remote add heroku git@heroku.com:%s.git", h.App))

	switch h.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to Heroku.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'adding build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push heroku HEAD:master --force"))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push heroku $COMMIT:master"))
	}
}
