package bitbucket

import (
	"os"

	"github.com/drone/drone/plugin/remote"
)

func init() {
	var cli = os.Getenv("BITBUCKET_CLIENT")
	var sec = os.Getenv("BITBUCKET_SECRET")
	if len(cli) == 0 || len(sec) == 0 {
		return
	}
	remote.Register(NewDefault(cli, sec))
}
