package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/router/middleware/token"
	"github.com/drone/drone/server"
	"github.com/drone/drone/static"
	"github.com/drone/drone/template"
)

func Load(middleware ...gin.HandlerFunc) http.Handler {

	e := gin.New()
	e.Use(gin.Recovery())

	e.SetHTMLTemplate(template.Load())
	e.StaticFS("/static", static.FileSystem())

	e.Use(header.NoCache)
	e.Use(header.Options)
	e.Use(header.Secure)
	e.Use(middleware...)
	e.Use(session.SetUser())
	e.Use(token.Refresh)

	e.GET("/", server.ShowIndex)
	e.GET("/repos", server.ShowAllRepos)
	e.GET("/login", server.ShowLogin)
	e.GET("/login/form", server.ShowLoginForm)
	e.GET("/logout", server.GetLogout)

	// TODO below will Go away with React UI
	settings := e.Group("/settings")
	{
		settings.Use(session.MustUser())
		settings.GET("/profile", server.ShowUser)
	}
	repo := e.Group("/repos/:owner/:name")
	{
		repo.Use(session.SetRepo())
		repo.Use(session.SetPerm())
		repo.Use(session.MustPull)

		repo.GET("", server.ShowRepo)
		repo.GET("/builds/:number", server.ShowBuild)
		repo.GET("/builds/:number/:job", server.ShowBuild)

		repo_settings := repo.Group("/settings")
		{
			repo_settings.GET("", session.MustPush, server.ShowRepoConf)
			repo_settings.GET("/encrypt", session.MustPush, server.ShowRepoEncrypt)
			repo_settings.GET("/badges", server.ShowRepoBadges)
		}
	}
	// TODO above will Go away with React UI

	user := e.Group("/api/user")
	{
		user.Use(session.MustUser())
		user.GET("", server.GetSelf)
		user.GET("/feed", server.GetFeed)
		user.GET("/repos", server.GetRepos)
		user.GET("/repos/remote", server.GetRemoteRepos)
		user.POST("/token", server.PostToken)
		user.DELETE("/token", server.DeleteToken)
	}

	users := e.Group("/api/users")
	{
		users.Use(session.MustAdmin())
		users.GET("", server.GetUsers)
		users.POST("", server.PostUser)
		users.GET("/:login", server.GetUser)
		users.PATCH("/:login", server.PatchUser)
		users.DELETE("/:login", server.DeleteUser)
	}

	repos := e.Group("/api/repos/:owner/:name")
	{
		repos.POST("", server.PostRepo)

		repo := repos.Group("")
		{
			repo.Use(session.SetRepo())
			repo.Use(session.SetPerm())
			repo.Use(session.MustPull)

			repo.GET("", server.GetRepo)
			repo.GET("/builds", server.GetBuilds)
			repo.GET("/builds/:number", server.GetBuild)
			repo.GET("/logs/:number/:job", server.GetBuildLogs)
			repo.POST("/sign", session.MustPush, server.Sign)

			repo.POST("/secrets", session.MustPush, server.PostSecret)
			repo.DELETE("/secrets/:secret", session.MustPush, server.DeleteSecret)

			// requires push permissions
			repo.PATCH("", session.MustPush, server.PatchRepo)
			repo.DELETE("", session.MustPush, server.DeleteRepo)

			repo.POST("/builds/:number", session.MustPush, server.PostBuild)
			repo.DELETE("/builds/:number/:job", session.MustPush, server.DeleteBuild)
		}
	}

	badges := e.Group("/api/badges/:owner/:name")
	{
		badges.GET("/status.svg", server.GetBadge)
		badges.GET("/cc.xml", server.GetCC)
	}

	e.POST("/hook", server.PostHook)
	e.POST("/api/hook", server.PostHook)

	stream := e.Group("/api/stream")
	{
		stream.Use(session.SetRepo())
		stream.Use(session.SetPerm())
		stream.Use(session.MustPull)

		stream.GET("/:owner/:name", server.GetRepoEvents)
		stream.GET("/:owner/:name/:build/:number", server.GetStream)
	}

	auth := e.Group("/authorize")
	{
		auth.GET("", server.GetLogin)
		auth.POST("", server.GetLogin)
		auth.POST("/token", server.GetLoginToken)
	}

	builds := e.Group("/api/builds")
	{
		builds.Use(session.MustAdmin())
		builds.GET("", server.GetBuildQueue)
	}

	agents := e.Group("/api/agents")
	{
		agents.Use(session.MustAdmin())
		agents.GET("", server.GetAgents)
	}

	queue := e.Group("/api/queue")
	{
		queue.Use(session.AuthorizeAgent)
		queue.POST("/pull", server.Pull)
		queue.POST("/pull/:os/:arch", server.Pull)
		queue.POST("/wait/:id", server.Wait)
		queue.POST("/stream/:id", server.Stream)
		queue.POST("/status/:id", server.Update)
		queue.POST("/ping", server.Ping)
	}

	// DELETE THESE
	// gitlab := e.Group("/gitlab/:owner/:name")
	// {
	// 	gitlab.Use(session.SetRepo())
	// 	gitlab.GET("/commits/:sha", GetCommit)
	// 	gitlab.GET("/pulls/:number", GetPullRequest)
	//
	// 	redirects := gitlab.Group("/redirect")
	// 	{
	// 		redirects.GET("/commits/:sha", RedirectSha)
	// 		redirects.GET("/pulls/:number", RedirectPullRequest)
	// 	}
	// }

	// bots := e.Group("/bots")
	// {
	// 	bots.Use(session.MustUser())
	// 	bots.POST("/slack", Slack)
	// 	bots.POST("/slack/:command", Slack)
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
