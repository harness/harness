package middleware

import (
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/bitbucket"
	"github.com/drone/drone/remote/github"
	"github.com/drone/drone/remote/gitlab"
	"github.com/drone/drone/remote/gogs"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ianschenck/envflag"
	"github.com/drone/drone/remote/bitbucketserver"
)

var (
	driver = envflag.String("REMOTE_DRIVER", "", "")
	config = envflag.String("REMOTE_CONFIG", "", "")
)

// Remote is a middleware function that initializes the Remote and attaches to
// the context of every http.Request.
func Remote() gin.HandlerFunc {

	logrus.Infof("using remote driver %s", *driver)
	logrus.Infof("using remote config %s", *config)

	var remote_ remote.Remote
	switch *driver {
	case "github":
		remote_ = github.Load(*config)
	case "bitbucket":
		remote_ = bitbucket.Load(*config)
	case "gogs":
		remote_ = gogs.Load(*config)
	case "gitlab":
		remote_ = gitlab.Load(*config)
	case "bitbucketserver":
		remote_ = bitbucketserver.Load(*config)
	default:
		logrus.Fatalln("remote configuraiton not found")
	}

	return func(c *gin.Context) {
		remote.ToContext(c, remote_)
		c.Next()
	}
}
