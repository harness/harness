package github

import (
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/model"
)

func init() {
	remote.Register(model.RemoteGithub, plugin)
	remote.Register(model.RemoteGithubEnterprise, plugin)
}

func plugin(remote *model.Remote) remote.Remote {
	return &Github{
		URL:     remote.Url,
		API:     remote.Api,
		Client:  remote.Client,
		Secret:  remote.Secret,
		Enabled: remote.Open,
	}
}
