package publish

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/plugin/publish/npm"
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

// Publish stores the configuration details
// for publishing build artifacts when
// a Build has succeeded
type Publish struct {
	S3     *S3      `yaml:"s3,omitempty"`
	Swift  *Swift   `yaml:"swift,omitempty"`
	PyPI   *PyPI    `yaml:"pypi,omitempty"`
	NPM    *npm.NPM `yaml:"npm,omitempty"`
	Docker *Docker  `yaml:"docker,omitempty"`
}

func (p *Publish) Write(f *buildfile.Buildfile, r *repo.Repo) {
	// S3
	if p.S3 != nil && match(p.S3.GetCondition(), r) {
		p.S3.Write(f)
	}

	// Swift
	if p.Swift != nil && match(p.Swift.GetCondition(), r) {
		p.Swift.Write(f)
	}

	// PyPI
	if p.PyPI != nil && match(p.PyPI.GetCondition(), r) {
		p.PyPI.Write(f)
	}

	// NPM
	if p.NPM != nil && match(p.NPM.GetCondition(), r) {
		p.NPM.Write(f)
	}

	// Docker
	if p.Docker != nil && (len(p.Docker.Branch) == 0 || (len(p.Docker.Branch) > 0 && r.Branch == p.Docker.Branch)) {
		p.Docker.Write(f, r)
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
