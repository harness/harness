package deploy

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

type Git struct {
	Target string `yaml:"target,omitempty"`
	Force  bool   `yaml:"force,omitempty"`
	Branch string `yaml:"branch,omitempty"`
}

func (g *Git) Write(f *buildfile.Buildfile) {
	// get the current commit hash
	f.WriteCmdSilent("COMMIT=$(git rev-parse HEAD)")

	// set the git user and email based on the individual
	// that made the commit.
	f.WriteCmdSilent("git config --global user.name $(git --no-pager log -1 --pretty=format:'%an')")
	f.WriteCmdSilent("git config --global user.email $(git --no-pager log -1 --pretty=format:'%ae')")

	// add target as a git remote
	f.WriteCmd(fmt.Sprintf("git remote add deploy %s", g.Target))

	destinationBranch := g.Branch
	if destinationBranch == "" {
		destinationBranch = "master"
	}

	switch g.Force {
	case true:
		// this is useful when the there are artifacts generated
		// by the build script, such as less files converted to css,
		// that need to be deployed to git remote.
		f.WriteCmd(fmt.Sprintf("git add -A"))
		f.WriteCmd(fmt.Sprintf("git commit -m 'add build artifacts'"))
		f.WriteCmd(fmt.Sprintf("git push deploy $COMMIT:%s --force", destinationBranch))
	case false:
		// otherwise we just do a standard git push
		f.WriteCmd(fmt.Sprintf("git push deploy $COMMIT:%s", destinationBranch))
	}
}
