package gitlab

import (
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/model"
)

func init() {
	remote.Register(model.RemoteGitlab, plugin)
}

func plugin(remote *model.Remote) remote.Remote {
	return &Gitlab{
		URL:     remote.Url,
		Enabled: remote.Open,
	}
}
