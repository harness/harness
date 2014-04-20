package deploy

import (
    "fmt"
    "github.com/drone/drone/pkg/build/buildfile"
)

type CloudFoundry struct {
    Target string `yaml:"target,omitempty"`
    Username string `yaml:"username,omitempty"`
    Password string `yaml:"password,omitempty"`
    Org string `yaml:"org,omitempty"`
    Space string `yaml:"space,omitempty"`

    App string `yaml:"app,omitempty"`
}

func (cf *CloudFoundry) Write(f *buildfile.Buildfile) {
    // login
    loginCmd := "cf login -a %s -u %s -p %s"

    organization := cf.Org
    if organization != "" {
        loginCmd += fmt.Sprintf(" -o %s", organization)
    }

    space := cf.Space
    if space != "" {
        loginCmd += fmt.Sprintf(" -s %s", space)
    }

    f.WriteCmdSilent(fmt.Sprintf(loginCmd, cf.Target, cf.Username, cf.Password))

    // push app
    pushCmd := "cf push %s"
    f.WriteCmd(fmt.Sprintf(pushCmd, cf.App))
}
