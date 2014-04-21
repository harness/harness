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
    downloadCmd := "curl -sLO http://go-cli.s3-website-us-east-1.amazonaws.com/releases/latest/cf-cli_amd64.deb"
    installCmd  := "dpkg -i cf-cli_amd64.deb 1> /dev/null 2> /dev/null"

    // download and install the cf tool
    f.WriteCmdSilent(fmt.Sprintf("[ -f /usr/bin/sudo ] && sudo %s || %s", downloadCmd, downloadCmd))
    f.WriteCmdSilent(fmt.Sprintf("[ -f /usr/bin/sudo ] && sudo %s || %s", installCmd, installCmd))

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
