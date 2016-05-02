package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/api"
	"github.com/drone/drone/bus"
	"github.com/drone/drone/cache"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router/middleware"
	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/router/middleware/token"
	"github.com/drone/drone/static"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"github.com/drone/drone/template"
	"github.com/drone/drone/web"

	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
)

// Config defines system configuration parameters.
type Config struct {
	Open   bool            // Enables open registration
	Yaml   string          // Customize the Yaml configuration file name
	Secret string          // Secret token used to authenticate agents
	Admins map[string]bool // Administrative users
	Orgs   map[string]bool // Organization whitelist
}

// Server defines the server configuration.
type Server struct {
	Bus    bus.Bus
	Cache  cache.Cache
	Queue  queue.Queue
	Remote remote.Remote
	Stream stream.Stream
	Store  store.Store
	Config *Config
}

// Handler returns an http.Handler for servering Drone requests.
func (s *Server) Handler() http.Handler {

	e := gin.New()
	e.Use(gin.Recovery())

	e.SetHTMLTemplate(template.Load())
	e.StaticFS("/static", static.FileSystem())

	e.Use(header.NoCache)
	e.Use(header.Options)
	e.Use(header.Secure)
	e.Use(
		ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true),
		HandlerVersion(),
		HandlerQueue(s.Queue),
		HandlerStream(s.Stream),
		HandlerBus(s.Bus),
		HandlerCache(s.Cache),
		HandlerStore(s.Store),
		HandlerRemote(s.Remote),
		HandlerConfig(s.Config),
	)
	e.Use(session.SetUser())
	e.Use(token.Refresh)

	e.GET("/", web.ShowIndex)
	e.GET("/repos", web.ShowAllRepos)
	e.GET("/login", web.ShowLogin)
	e.GET("/login/form", web.ShowLoginForm)
	e.GET("/logout", GetLogout)

	// TODO below will Go away with React UI
	settings := e.Group("/settings")
	{
		settings.Use(session.MustUser())
		settings.GET("/profile", web.ShowUser)
	}
	repo := e.Group("/repos/:owner/:name")
	{
		repo.Use(session.SetRepo())
		repo.Use(session.SetPerm())
		repo.Use(session.MustPull)

		repo.GET("", web.ShowRepo)
		repo.GET("/builds/:number", web.ShowBuild)
		repo.GET("/builds/:number/:job", web.ShowBuild)

		repo_settings := repo.Group("/settings")
		{
			repo_settings.GET("", session.MustPush, web.ShowRepoConf)
			repo_settings.GET("/encrypt", session.MustPush, web.ShowRepoEncrypt)
			repo_settings.GET("/badges", web.ShowRepoBadges)
		}
	}
	// TODO above will Go away with React UI

	user := e.Group("/api/user")
	{
		user.Use(session.MustUser())
		user.GET("", api.GetSelf)
		user.GET("/feed", api.GetFeed)
		user.GET("/repos", api.GetRepos)
		user.GET("/repos/remote", api.GetRemoteRepos)
		user.POST("/token", api.PostToken)
		user.DELETE("/token", api.DeleteToken)
	}

	users := e.Group("/api/users")
	{
		users.Use(session.MustAdmin())
		users.GET("", api.GetUsers)
		users.POST("", api.PostUser)
		users.GET("/:login", api.GetUser)
		users.PATCH("/:login", api.PatchUser)
		users.DELETE("/:login", api.DeleteUser)
	}

	repos := e.Group("/api/repos/:owner/:name")
	{
		repos.POST("", api.PostRepo)

		repo := repos.Group("")
		{
			repo.Use(session.SetRepo())
			repo.Use(session.SetPerm())
			repo.Use(session.MustPull)

			repo.GET("", api.GetRepo)
			repo.GET("/builds", api.GetBuilds)
			repo.GET("/builds/:number", api.GetBuild)
			repo.GET("/logs/:number/:job", api.GetBuildLogs)
			repo.POST("/sign", session.MustPush, api.Sign)

			repo.POST("/secrets", session.MustPush, api.PostSecret)
			repo.DELETE("/secrets/:secret", session.MustPush, api.DeleteSecret)

			// requires push permissions
			repo.PATCH("", session.MustPush, api.PatchRepo)
			repo.DELETE("", session.MustPush, api.DeleteRepo)

			repo.POST("/builds/:number", session.MustPush, api.PostBuild)
			repo.DELETE("/builds/:number/:job", session.MustPush, api.DeleteBuild)
		}
	}

	badges := e.Group("/api/badges/:owner/:name")
	{
		badges.GET("/status.svg", web.GetBadge)
		badges.GET("/cc.xml", web.GetCC)
	}

	e.POST("/hook", web.PostHook)
	e.POST("/api/hook", web.PostHook)

	stream := e.Group("/api/stream")
	{
		stream.Use(session.SetRepo())
		stream.Use(session.SetPerm())
		stream.Use(session.MustPull)

		stream.GET("/:owner/:name", web.GetRepoEvents)
		stream.GET("/:owner/:name/:build/:number", web.GetStream)
	}

	auth := e.Group("/authorize")
	{
		auth.GET("", GetLogin)
		auth.POST("", GetLogin)
		auth.POST("/token", GetLoginToken)
	}

	queue := e.Group("/api/queue")
	{
		queue.Use(middleware.AgentMust())
		queue.POST("/pull", api.Pull)
		queue.POST("/pull/:os/:arch", api.Pull)
		queue.POST("/wait/:id", api.Wait)
		queue.POST("/stream/:id", api.Stream)
		queue.POST("/status/:id", api.Update)
	}

	// DELETE THESE
	// gitlab := e.Group("/gitlab/:owner/:name")
	// {
	// 	gitlab.Use(session.SetRepo())
	// 	gitlab.GET("/commits/:sha", web.GetCommit)
	// 	gitlab.GET("/pulls/:number", web.GetPullRequest)
	//
	// 	redirects := gitlab.Group("/redirect")
	// 	{
	// 		redirects.GET("/commits/:sha", web.RedirectSha)
	// 		redirects.GET("/pulls/:number", web.RedirectPullRequest)
	// 	}
	// }

	// bots := e.Group("/bots")
	// {
	// 	bots.Use(session.MustUser())
	// 	bots.POST("/slack", web.Slack)
	// 	bots.POST("/slack/:command", web.Slack)
	// }

	return normalize(e)
}

// THIS HACK JOB IS GOING AWAY SOON.
//
// normalize is a helper function to work around the following
// issue with gin. https://github.com/gin-gonic/gin/issues/388
func normalize(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		parts := strings.Split(r.URL.Path, "/")[1:]
		switch parts[0] {
		case "settings", "bots", "repos", "api", "login", "logout", "", "authorize", "hook", "static", "gitlab":
			// no-op
		default:

			if len(parts) > 2 && parts[2] != "settings" {
				parts = append(parts[:2], append([]string{"builds"}, parts[2:]...)...)
			}

			// prefix the URL with /repo so that it
			// can be effectively routed.
			parts = append([]string{"", "repos"}, parts...)

			// reconstruct the path
			r.URL.Path = strings.Join(parts, "/")
		}

		h.ServeHTTP(w, r)
	})
}
