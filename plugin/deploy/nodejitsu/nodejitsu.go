package nodejitsu

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Nodejitsu struct {
	User  string `yaml:"user,omitempty"`
	Token string `yaml:"token,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (n *Nodejitsu) Write(f *buildfile.Buildfile) {
	if len(n.Token) == 0 || len(n.User) == 0 {
		return
	}

	f.WriteEnv("username", n.User)
	f.WriteEnv("apiToken", n.Token)

	// Install the jitsu command line interface then
	// deploy the configured app.
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || npm install -g jitsu")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo npm install -g jitsu")
	f.WriteCmd("jitsu deploy")
}

func (n *Nodejitsu) GetCondition() *condition.Condition {
	return n.Condition
}
