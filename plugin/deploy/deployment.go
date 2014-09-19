package deploy

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

// Deploy stores the configuration details
// for deploying build artifacts when
// a Build has succeeded
type Deploy struct {
	AppFog       *AppFog       `yaml:"appfog,omitempty"`
	CloudControl *CloudControl `yaml:"cloudcontrol,omitempty"`
	CloudFoundry *CloudFoundry `yaml:"cloudfoundry,omitempty"`
	EngineYard   *EngineYard   `yaml:"engineyard,omitempty"`
	Git          *Git          `yaml:"git,omitempty"`
	Heroku       *Heroku       `yaml:"heroku,omitempty"`
	Modulus      *Modulus      `yaml:"modulus,omitempty"`
	Nodejitsu    *Nodejitsu    `yaml:"nodejitsu,omitempty"`
	Openshift    *Openshift    `yaml:"openshift,omitempty"`
	SSH          *SSH          `yaml:"ssh,omitempty"`
	Tsuru        *Tsuru        `yaml:"tsuru,omitempty"`
	Bash         *Bash         `yaml:"bash,omitempty"`
}

func (d *Deploy) Write(f *buildfile.Buildfile, r *repo.Repo) {

	if d.CloudFoundry != nil && match(d.CloudFoundry.GetCondition(), r) {
		d.CloudFoundry.Write(f)
	}
	if d.Git != nil && match(d.Git.GetCondition(), r) {
		d.Git.Write(f)
	}
	if d.Heroku != nil && match(d.Heroku.GetCondition(), r) {
		d.Heroku.Write(f)
	}
	if d.Modulus != nil && match(d.Modulus.GetCondition(), r) {
		d.Modulus.Write(f)
	}
	if d.Nodejitsu != nil && match(d.Nodejitsu.GetCondition(), r) {
		d.Nodejitsu.Write(f)
	}
	if d.SSH != nil && match(d.SSH.GetCondition(), r) {
		d.SSH.Write(f)
	}
	if d.Tsuru != nil && match(d.Tsuru.GetCondition(), r) {
		d.Tsuru.Write(f)
	}
	if d.Bash != nil && match(d.Bash.GetCondition(), r) {
		d.Bash.Write(f)
	}
}

func match(c *condition.Condition, r *repo.Repo) bool {
	switch {
	case c == nil:
		return true
	case !c.MatchTag(r.Tag):
		return false
	case !c.MatchBranch(r.Branch):
		return false
	case !c.MatchOwner(r.Name):
		return false
	case !c.MatchPullRequest(r.PR):
		return false
	}
	return true
}
