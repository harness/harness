package deploy

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

type Modulus struct {
	Project string `yaml:"project,omitempty"`
	Token   string `yaml:"token,omitempty"`
}

func (m *Modulus) Write(f *buildfile.Buildfile) {
	f.WriteEnv("MODULUS_TOKEN", m.Token)

	// Install the Modulus command line interface then deploy the configured
	// project.
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || npm install -g modulus")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo npm install -g modulus")
	f.WriteCmd(fmt.Sprintf("modulus deploy -p '%s'", m.Project))
}
