package publish

import (
	"fmt"
	"strings"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

import ()

type Github struct {
	// Script is an optional list of commands to run to prepare for a release.
	Script []string `yaml:"script"`

	// Artifacts is a list of files or directories to release.
	Artifacts []string `yaml:"artifacts"`

	// Tag is the name of the tag to create for this release.
	Tag string `yaml:"tag"`

	// Name is the name of the release. Defaults to tag.
	Name string `yaml:"name"`

	// Description describes the release. Defaults to empty string.
	Description string `yaml:"description"`

	// Draft is an identifier on a Github release.
	Draft bool `yaml:"draft"`

	// Prerelease is an identifier on a Github release.
	Prerelease bool `yaml:"prerelease"`

	// Token is the Github token to use when publishing the release.
	Token string `yaml:"token"`

	// User is the Github user for the repository you'd like to publish to.
	User string `yaml:"user"`

	// Repo is the name of the Github repostiory you like to publish to.
	Repo string `yaml:"repo"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

// Write adds commands to run that will publish a Github release.
func (g *Github) Write(f *buildfile.Buildfile) {
	if len(g.Artifacts) == 0 || g.Tag == "" || g.Token == "" || g.User == "" || g.Repo == "" {
		f.WriteCmdSilent(`echo -e "Github Plugin: Missing argument(s)"\n\n`)
		if len(g.Artifacts) == 0 {
			f.WriteCmdSilent(`echo -e "\tartifacts not defined in yaml config" && false`)
		}
		if g.Tag == "" {
			f.WriteCmdSilent(`echo -e "\ttag not defined in yaml config" && false`)
		}
		if g.Token == "" {
			f.WriteCmdSilent(`echo -e "\ttoken not defined in yaml config" && false`)
		}
		if g.User == "" {
			f.WriteCmdSilent(`echo -e "\tuser not defined in yaml config" && false`)
		}
		if g.Repo == "" {
			f.WriteCmdSilent(`echo -e "\trepo not defined in yaml config" && false`)
		}
		return
	}

	// Default name is tag
	if g.Name == "" {
		g.Name = g.Tag
	}

	for _, cmd := range g.Script {
		f.WriteCmd(cmd)
	}

	f.WriteEnv("GITHUB_TOKEN", g.Token)

	// Install github-release
	f.WriteCmd("curl -L -o /tmp/github-release.tar.bz2 https://github.com/aktau/github-release/releases/download/v0.5.2/linux-amd64-github-release.tar.bz2")
	f.WriteCmd("tar jxf /tmp/github-release.tar.bz2 -C /tmp/ && sudo mv /tmp/bin/linux/amd64/github-release /usr/local/bin/github-release")

	// Create the release. Ignore 422 errors, which indicate the tag has already been created.
	// Doing otherwise would create the expectation that every commit should be tagged and released,
	// which is not the norm.
	draftStr := ""
	if g.Draft {
		draftStr = "--draft"
	}
	prereleaseStr := ""
	if g.Prerelease {
		prereleaseStr = "--pre-release"
	}
	f.WriteCmd(fmt.Sprintf(`
result=$(github-release release -u %s -r %s -t %s -n "%s" -d "%s" %s %s || true)
if [[ $result == *422* ]]; then
  echo -e "Release already exists for this tag.";
  exit 0
elif [[ $result == "" ]]; then
  echo -e "Release created.";
else
  echo -e "Error creating release: $result"
  exit 1
fi
`, g.User, g.Repo, g.Tag, g.Name, g.Description, draftStr, prereleaseStr))

	// Upload files
	artifactStr := strings.Join(g.Artifacts, " ")
	f.WriteCmd(fmt.Sprintf(`
for f in %s; do
    # treat directories and files differently
    if [ -d $f ]; then
        for ff in $(ls $f); do
            echo -e "uploading $ff"
            github-release upload -u %s -r %s -t %s -n $ff -f $f/$ff
        done
    elif [ -f $f ]; then
        echo -e "uploading $f"
        github-release upload -u %s -r %s -t %s -n $f -f $f
    else
        echo -e "$f is not a file or directory"
        exit 1
    fi
done
`, artifactStr, g.User, g.Repo, g.Tag, g.User, g.Repo, g.Tag))
}

func (g *Github) GetCondition() *condition.Condition {
	return g.Condition
}
