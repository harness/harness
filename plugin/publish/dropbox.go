package publish

import (
	"fmt"
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
	"strings"
)

type Dropbox struct {
	AccessToken string `yaml:"access_token,omitempty"`

	Source string `yaml:"source,omitempty"`
	Target string `yaml:"target,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (d *Dropbox) Write(f *buildfile.Buildfile) {

	if len(d.AccessToken) == 0 || len(d.Source) == 0 || len(d.Target) == 0 {
		return
	}
	if strings.HasPrefix(d.Target, "/") {
		d.Target = d.Target[1:]
	}

	f.WriteCmdSilent("echo 'publishing to Dropbox ...'")

	cmd := "curl --upload-file %s -H \"Authorization: Bearer %s\" \"https://api-content.dropbox.com/1/files_put/auto/%s?overwrite=true\""
	f.WriteCmd(fmt.Sprintf(cmd, d.Source, d.AccessToken, d.Target))

}

func (d *Dropbox) GetCondition() *condition.Condition {
	return d.Condition
}
