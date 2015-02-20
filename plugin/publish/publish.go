package publish

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/plugin/publish/bintray"
	"github.com/drone/drone/plugin/publish/npm"
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

// Publish stores the configuration details
// for publishing build artifacts when
// a Build has succeeded
type Publish struct {
	Azure   *Azure           `yaml:"azure,omitempty"`
	S3      *S3              `yaml:"s3,omitempty"`
	Swift   *Swift           `yaml:"swift,omitempty"`
	PyPI    *PyPI            `yaml:"pypi,omitempty"`
	NPM     *npm.NPM         `yaml:"npm,omitempty"`
	Docker  *Docker          `yaml:"docker,omitempty"`
	Github  *Github          `yaml:"github,omitempty"`
	Dropbox *Dropbox         `yaml:"dropbox,omitempty"`
	Bintray *bintray.Bintray `yaml:"bintray,omitempty"`
}

func (p *Publish) Write(f *buildfile.Buildfile, r *repo.Repo) {
	// Azure
	if p.Azure != nil && match(p.Azure.GetCondition(), r) {
		p.Azure.Write(f)
	}

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

	// Github
	if p.Github != nil && match(p.Github.GetCondition(), r) {
		p.Github.Write(f)
	}

	// Docker
	if p.Docker != nil && match(p.Docker.GetCondition(), r) {
		p.Docker.Write(f)
	}

	// Dropbox
	if p.Dropbox != nil && match(p.Dropbox.GetCondition(), r) {
		p.Dropbox.Write(f)
	}

	// Bintray
	if p.Bintray != nil && match(p.Bintray.GetCondition(), r) {
		p.Bintray.Write(f)
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
