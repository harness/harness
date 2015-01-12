package bintray

import (
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Bintray struct {
	Username string    `yaml:"username"`
	ApiKey   string    `yaml:"api_key"`
	Packages []Package `yaml:"packages"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (b *Bintray) Write(f *buildfile.Buildfile) {
	var cmd string

	// Validate Username, ApiKey, Packages
	if len(b.Username) == 0 || len(b.ApiKey) == 0 || len(b.Packages) == 0 {
		f.WriteCmdSilent(`echo -e "Bintray Plugin: Missing argument(s)\n\n"`)

		if len(b.Username) == 0 {
			f.WriteCmdSilent(`echo -e "\tusername not defined in yaml config"`)
		}

		if len(b.ApiKey) == 0 {
			f.WriteCmdSilent(`echo -e "\tapi_key not defined in yaml config"`)
		}

		if len(b.Packages) == 0 {
			f.WriteCmdSilent(`echo -e "\tpackages not defined in yaml config"`)
		}

		f.WriteCmdSilent("exit 1")

		return
	}

	for _, pkg := range b.Packages {
		pkg.Write(b.Username, b.ApiKey, f)
	}

	f.WriteCmd(cmd)

}

func (b *Bintray) GetCondition() *condition.Condition {
	return b.Condition
}
