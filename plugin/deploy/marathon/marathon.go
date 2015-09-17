package marathon

import (
	"fmt"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Marathon struct {
	//Hostname for the Marathon Master
	Host string `yaml:"host,omitempty"`
	App  string `yaml:"app,omitempty"`

	// The app config for marathon
	//https://mesosphere.github.io/marathon/docs/rest-api.html#post-v2-apps
	// Examples:
	//    /path/to/file
	//    /path/to/*.txt
	//    /path/to/*/*.txt
	//    /path/to/**
	ConfigFile string               `yaml:"config_file,omitempty"`
	Condition  *condition.Condition `yaml:"when,omitempty"`
}

func (m *Marathon) Write(f *buildfile.Buildfile) {
	// debugging purposes so we can see if / where something is failing
	f.WriteCmdSilent("echo 'deploying to Marathon ...'")

	put := fmt.Sprintf(
		"curl -X PUT -d @%s http://%s/v2/apps/%s --header \"Content-Type:application/json\"",
		m.ConfigFile,
		m.Host,
		m.App,
	)
	f.WriteCmdSilent(put)
}

func (m *Marathon) GetCondition() *condition.Condition {
	return m.Condition
}
