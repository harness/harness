package publish

import (
	"fmt"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Azure struct {
	StorageAccount   string `yaml:"storage_account,omitempty"`
	StorageAccessKey string `yaml:"storage_access_key,omitempty"`
	StorageContainer string `yaml:"storage_container,omitempty"`

	// Uploads file indicated by Source to file
	// indicated by Target. Only individual file names
	// are supported by Source and Target
	Source string `yaml:"source,omitempty"`
	Target string `yaml:"target"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (a *Azure) Write(f *buildfile.Buildfile) {
	if len(a.StorageAccount) == 0 || len(a.StorageAccessKey) == 0 || len(a.StorageContainer) == 0 || len(a.Source) == 0 {
		return
	}

	f.WriteCmdSilent("echo 'publishing to Azure Storage ...'")

	// install Azure xplat CLI
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || npm install -g azure-cli 1> /dev/null 2> /dev/null")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo npm install -g azure-cli 1> /dev/null 2> /dev/null")

	f.WriteEnv("AZURE_STORAGE_ACCOUNT", a.StorageAccount)
	f.WriteEnv("AZURE_STORAGE_ACCESS_KEY", a.StorageAccessKey)

	// if target isn't specified, set to source
	if len(a.Target) == 0 {
		a.Target = a.Source
	}

	f.WriteCmd(fmt.Sprintf(`azure storage blob upload --container %s %s %s`, a.StorageContainer, a.Source, a.Target))
}

func (a *Azure) GetCondition() *condition.Condition {
	return a.Condition
}
