package publish

import (
	"github.com/drone/drone/pkg/build/buildfile"
	"github.com/drone/drone/pkg/build/repo"
)

// Publish stores the configuration details
// for publishing build artifacts when
// a Build has succeeded
type Publish struct {
	S3    *S3    `yaml:"s3,omitempty"`
	Swift *Swift `yaml:"swift,omitempty"`
	PyPI  *PyPI  `yaml:"pypi,omitempty"`
	NPM   *NPM   `yaml:"npm,omitempty"`
}

func (p *Publish) Write(f *buildfile.Buildfile, r *repo.Repo) {
	// S3
	if p.S3 != nil && (len(p.S3.Branch) == 0 || (len(p.S3.Branch) > 0 && r.Branch == p.S3.Branch)) {
		p.S3.Write(f)
	}

	// Swift
	if p.Swift != nil && (len(p.Swift.Branch) == 0 || (len(p.Swift.Branch) > 0 && r.Branch == p.Swift.Branch)) {
		p.Swift.Write(f)
	}

	// PyPI
	if p.PyPI != nil && (len(p.PyPI.Branch) == 0 || (len(p.PyPI.Branch) > 0 && r.Branch == p.PyPI.Branch)) {
		p.PyPI.Write(f)
	}

	// NPM
	if p.NPM != nil && (len(p.NPM.Branch) == 0 || (len(p.NPM.Branch) > 0 && r.Branch == p.NPM.Branch)) {
		p.NPM.Write(f)
	}
}
