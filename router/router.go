package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/api"
	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/router/middleware/location"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/router/middleware/token"
	"github.com/drone/drone/static"
	"github.com/drone/drone/template"
	"github.com/drone/drone/web"
)

func Load(middleware ...gin.HandlerFunc) http.Handler {
	e := gin.Default()
	e.SetHTMLTemplate(template.Load())
	e.StaticFS("/static", static.FileSystem())

	e.Use(location.Resolve)
	e.Use(header.NoCache)
	e.Use(header.Options)
	e.Use(header.Secure)
	e.Use(middleware...)
	e.Use(session.SetUser())
	e.Use(token.Refresh)

	e.GET("/", web.ShowIndex)
	e.GET("/repos", web.ShowAllRepos)
	e.GET("/login", web.ShowLogin)
	e.GET("/login/form", web.ShowLoginForm)
	e.GET("/logout", web.GetLogout)

	settings := e.Group("/settings")
	{
		settings.Use(session.MustUser())
		settings.GET("/profile", web.ShowUser)
		settings.GET("/people", session.MustAdmin(), web.ShowUsers)
		settings.GET("/nodes", session.MustAdmin(), web.ShowNodes)
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

	nodes := e.Group("/api/nodes")
	{
		nodes.Use(session.MustAdmin())
		nodes.GET("", api.GetNodes)
		nodes.POST("", api.PostNode)
		nodes.DELETE("/:node", api.DeleteNode)
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
			repo.GET("/key", api.GetRepoKey)
			repo.POST("/key", api.PostRepoKey)
			repo.GET("/builds", api.GetBuilds)
			repo.GET("/builds/:number", api.GetBuild)
			repo.GET("/logs/:number/:job", api.GetBuildLogs)
			repo.POST("/sign", session.MustPush, api.Sign)

			repo.POST("/secrets", session.MustPush, api.PostSecret)
			repo.DELETE("/secrets/:secret", session.MustPush, api.DeleteSecret)

			// requires authenticated user
			repo.POST("/encrypt", session.MustUser(), api.PostSecure)

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
		auth.GET("", web.GetLogin)
		auth.POST("", web.GetLogin)
		auth.POST("/token", web.GetLoginToken)
	}

	gitlab := e.Group("/gitlab/:owner/:name")
	{
		gitlab.Use(session.SetRepo())
		gitlab.GET("/commits/:sha", web.GetCommit)
		gitlab.GET("/pulls/:number", web.GetPullRequest)

		redirects := gitlab.Group("/redirect")
		{
			redirects.GET("/commits/:sha", web.RedirectSha)
			redirects.GET("/pulls/:number", web.RedirectPullRequest)
		}
	}

	return normalize(e)
}

// normalize is a helper function to work around the following
// issue with gin. https://github.com/gin-gonic/gin/issues/388
func normalize(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		parts := strings.Split(r.URL.Path, "/")[1:]
		switch parts[0] {
		case "settings", "repos", "api", "login", "logout", "", "authorize", "hook", "static", "gitlab":
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
