package deploy

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"

	"github.com/drone/drone/plugin/deploy/deis"
	"github.com/drone/drone/plugin/deploy/git"
	"github.com/drone/drone/plugin/deploy/heroku"
	"github.com/drone/drone/plugin/deploy/marathon"
	"github.com/drone/drone/plugin/deploy/modulus"
	"github.com/drone/drone/plugin/deploy/nodejitsu"
	"github.com/drone/drone/plugin/deploy/tsuru"
)

// Deploy stores the configuration details
// for deploying build artifacts when
// a Build has succeeded
type Deploy struct {
	CloudFoundry *CloudFoundry        `yaml:"cloudfoundry,omitempty"`
	Git          *git.Git             `yaml:"git,omitempty"`
	Heroku       *heroku.Heroku       `yaml:"heroku,omitempty"`
	Deis         *deis.Deis           `yaml:"deis,omitempty"`
	Modulus      *modulus.Modulus     `yaml:"modulus,omitempty"`
	Nodejitsu    *nodejitsu.Nodejitsu `yaml:"nodejitsu,omitempty"`
	SSH          *SSH                 `yaml:"ssh,omitempty"`
	Tsuru        *tsuru.Tsuru         `yaml:"tsuru,omitempty"`
	Bash         *Bash                `yaml:"bash,omitempty"`
	Marathon     *marathon.Marathon   `yaml:"marathon,omitempty"`
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
	if d.Deis != nil && match(d.Deis.GetCondition(), r) {
		d.Deis.Write(f)
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
	if d.Marathon != nil && match(d.Marathon.GetCondition(), r) {
		d.Marathon.Write(f)
	}
}

func match(c *condition.Condition, r *repo.Repo) bool {
	switch {
	case c == nil:
		return true
	case !c.MatchBranch(r.Branch):
		return false
	case !c.MatchOwner(r.Name):
		return false
	case !c.MatchPullRequest(r.PR):
		return false
	}
	return true
}
