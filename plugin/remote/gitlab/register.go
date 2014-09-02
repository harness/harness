package gitlab

import (
	"os"

	"github.com/drone/drone/plugin/remote"
)

// registers the Gitlab plugin
func init() {
	var url = os.Getenv("GITLAB_URL")
	if len(url) == 0 {
		return
	}
	remote.Register(New(url))
}
