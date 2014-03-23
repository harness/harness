package deploy

import (
	"github.com/drone/drone/pkg/build/buildfile"
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
}

func (d *Deploy) Write(f *buildfile.Buildfile) {
	if d.AppFog != nil {
		d.AppFog.Write(f)
	}
	if d.CloudControl != nil {
		d.CloudControl.Write(f)
	}
	if d.CloudFoundry != nil {
		d.CloudFoundry.Write(f)
	}
	if d.EngineYard != nil {
		d.EngineYard.Write(f)
	}
	if d.Git != nil {
		d.Git.Write(f)
	}
	if d.Heroku != nil {
		d.Heroku.Write(f)
	}
	if d.Modulus != nil {
		d.Modulus.Write(f)
	}
	if d.Nodejitsu != nil {
		d.Nodejitsu.Write(f)
	}
	if d.Openshift != nil {
		d.Openshift.Write(f)
	}
	if d.SSH != nil {
		d.SSH.Write(f)
	}
	if d.Tsuru != nil {
		d.Tsuru.Write(f)
	}
}
