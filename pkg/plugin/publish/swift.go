package publish

import (
	"fmt"
	"strings"

	"github.com/drone/drone/pkg/build/buildfile"
)

type Swift struct {
	// Username for authentication
	Username string `yaml:"username,omitempty"`

	// Password for authentication
	// With Rackspace this is usually an API Key
	Password string `yaml:"password,omitempty"`

	// Container to upload files to
	Container string `yaml:"container,omitempty"`

	// Base API version URL to authenticate against
	// Rackspace: https://identity.api.rackspacecloud.com/v2.0
	AuthURL string `yaml:"auth_url,omitempty"`

	// Region to communicate with, in a generic OpenStack install
	// this may be RegionOne
	Region string `yaml:"region,omitempty"`

	// Source file or directory to upload, if source is a directory,
	// upload the contents of the directory
	Source string `yaml:"source,omitempty"`

	// Destination to write the file(s) to. Should contain the full
	// object name if source is a file
	Target string `yaml:"target,omitempty"`

	Branch string `yaml:"branch,omitempty"`
}

func (s *Swift) Write(f *buildfile.Buildfile) {
	var target string
	// All options are required, so ensure they are present
	if len(s.Username) == 0 || len(s.Password) == 0 || len(s.AuthURL) == 0 || len(s.Region) == 0 || len(s.Source) == 0 || len(s.Container) == 0 {
		f.WriteCmdSilent(`echo "Swift: Missing argument(s)"`)
		return
	}

	// If a target was provided, prefix it with a /
	if len(s.Target) > 0 {
		target = fmt.Sprintf("/%s", strings.TrimPrefix(s.Target, "/"))
	}

	// debugging purposes so we can see if / where something is failing
	f.WriteCmdSilent(`echo "Swift: Publishing..."`)

	// install swiftly using PIP
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || pip install swiftly 1> /dev/null 2> /dev/null")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo pip install swiftly 1> /dev/null 2> /dev/null")

	// Write out environment variables
	f.WriteEnv("SWIFTLY_AUTH_URL", s.AuthURL)
	f.WriteEnv("SWIFTLY_AUTH_USER", s.Username)
	f.WriteEnv("SWIFTLY_AUTH_KEY", s.Password)
	f.WriteEnv("SWIFTLY_REGION", s.Region)

	f.WriteCmd(fmt.Sprintf(`swiftly put -i %s %s%s`, s.Source, s.Container, target))
}
