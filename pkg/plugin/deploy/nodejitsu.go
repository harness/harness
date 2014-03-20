package deploy

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

type Nodejitsu struct {
	App   string `yaml:"app,omitempty"`
	User  string `yaml:"user,omitempty"`
	Token string `yaml:"token,omitempty"`
}

func (n *Nodejitsu) Write(f *buildfile.Buildfile) {
	f.WriteEnv("username", m.User)
	f.WriteEnv("apiToken", m.Token)

	// Install the jitsu command line interface then
	// deploy the configured app.
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || npm install -g jitsu")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo npm install -g jitsu")
	f.WriteCmd("jitsu deploy")
}
