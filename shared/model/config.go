package model

import (
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/plugin/remote/bitbucket"
	"github.com/drone/drone/plugin/remote/github"
	"github.com/drone/drone/plugin/remote/gitlab"
	"github.com/drone/drone/plugin/remote/stash"
	"github.com/drone/drone/plugin/smtp"
)

type Config struct {
	// Hostname of the server, eg drone.io
	//Host string `json:"host"`

	// Scheme of the server, eg https
	//Scheme string `json:"scheme"`

	// Registration with a value of True allows developers
	// to register themselves. If false, must be approved
	// or invited by the system administrator.
	Registration bool `json:"registration"`

	// SMTP stores configuration details for connecting with
	// and smtp server to send email notifications.
	SMTP *smtp.SMTP `json:"smtp"`

	// Bitbucket stores configuration details for communicating
	// with the bitbucket.org public cloud service.
	Bitbucket *bitbucket.Bitbucket `json:"bitbucket"`

	// Github stores configuration details for communicating
	// with the github.com public cloud service.
	Github *github.Github `json:"github"`

	// GithubEnterprise stores configuration details for
	// communicating with a private Github installation.
	GithubEnterprise *github.Github `json:"githubEnterprise"`

	// Gitlab stores configuration details for communicating
	// with a private gitlab installation.
	Gitlab *gitlab.Gitlab `json:"gitlab"`

	// Stash stores configuration details for communicating
	// with a private Atlassian Stash installation.
	Stash *stash.Stash `json:"stash"`
}

// GetRemote is a helper function that will return the
// remote plugin name based on the specified hostname.
func (c *Config) GetRemote(name string) remote.Remote {
	// first attempt to get the remote instance
	// by the unique plugin name (ie enterprise.github.com)
	switch name {
	case c.Github.GetName():
		return c.Github
	case c.Bitbucket.GetName():
		return c.Bitbucket
	case c.GithubEnterprise.GetName():
		return c.GithubEnterprise
	case c.Gitlab.GetName():
		return c.Gitlab
	case c.Stash.GetName():
		return c.Stash
	}

	// else attempt to get the remote instance
	// by the hostname (ie github.drone.io)
	switch {
	case c.Github.IsMatch(name):
		return c.Github
	case c.Bitbucket.IsMatch(name):
		return c.Bitbucket
	case c.GithubEnterprise.IsMatch(name):
		return c.GithubEnterprise
	case c.Gitlab.IsMatch(name):
		return c.Gitlab
	case c.Stash.IsMatch(name):
		return c.Stash
	}

	// else none found
	return nil
}

// GetClient is a helper function taht will return the named
// remote plugin client, used to interact with the remote system.
func (c *Config) GetClient(name, access, secret string) remote.Client {
	remote := c.GetRemote(name)
	if remote == nil {
		return nil
	}
	return remote.GetClient(access, secret)
}
