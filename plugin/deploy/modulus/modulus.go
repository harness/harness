package modulus

import (
	"fmt"
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Modulus struct {
	Project string `yaml:"project,omitempty"`
	Token   string `yaml:"token,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (m *Modulus) Write(f *buildfile.Buildfile) {
	if len(m.Token) == 0 || len(m.Project) == 0 {
		return
	}
	f.WriteEnv("MODULUS_TOKEN", m.Token)

	// Verify npm exists, otherwise we cannot install the
	// modulus command line utility.
	f.WriteCmdSilent("[ -f /usr/bin/npm ] || echo ERROR: npm is required for modulus.io deployments")
	f.WriteCmdSilent("[ -f /usr/bin/npm ] || exit 1")

	// Install the Modulus command line interface then deploy the configured
	// project.
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || npm install -g modulus")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo npm install -g modulus")
	f.WriteCmd(fmt.Sprintf("modulus deploy -p %q", m.Project))
}

func (m *Modulus) GetCondition() *condition.Condition {
	return m.Condition
}
