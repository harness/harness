package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/router/middleware/token"
	"github.com/drone/drone/server"
	"github.com/drone/drone/server/debug"
	"github.com/drone/drone/server/metrics"
	"github.com/drone/drone/server/template"
)

// Load loads the router
func Load(middleware ...gin.HandlerFunc) http.Handler {

	e := gin.New()
	e.Use(gin.Recovery())
	e.SetHTMLTemplate(template.T)

	ui := server.NewWebsite()
	for _, path := range ui.Routes() {
		e.GET(path, func(c *gin.Context) {
			ui.File(c.Writer, c.Request)
		})
	}

	e.Use(header.NoCache)
	e.Use(header.Options)
	e.Use(header.Secure)
	e.Use(middleware...)
	e.Use(session.SetUser())
	e.Use(token.Refresh)

	e.GET("/logout", server.GetLogout)
	e.NoRoute(func(c *gin.Context) {
		u := session.User(c)
		ui.Page(c.Writer, c.Request, u)
	})

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
			repo.GET("/logs/:number/:ppid/:proc", server.GetBuildLogs)

			// requires push permissions
			repo.GET("/secrets", session.MustPush, server.GetSecretList)
			repo.POST("/secrets", session.MustPush, server.PostSecret)
			repo.GET("/secrets/:secret", session.MustPush, server.GetSecret)
			repo.PATCH("/secrets/:secret", session.MustPush, server.PatchSecret)
			repo.DELETE("/secrets/:secret", session.MustPush, server.DeleteSecret)

			// requires push permissions
			repo.GET("/registry", session.MustPush, server.GetRegistryList)
			repo.POST("/registry", session.MustPush, server.PostRegistry)
			repo.GET("/registry/:registry", session.MustPush, server.GetRegistry)
			repo.PATCH("/registry/:registry", session.MustPush, server.PatchRegistry)
			repo.DELETE("/registry/:registry", session.MustPush, server.DeleteRegistry)

			// requires push permissions
			repo.PATCH("", session.MustPush, server.PatchRepo)
			repo.DELETE("", session.MustRepoAdmin(), server.DeleteRepo)
			repo.POST("/chown", session.MustRepoAdmin(), server.ChownRepo)
			repo.POST("/repair", session.MustRepoAdmin(), server.RepairRepo)

			repo.POST("/builds/:number", session.MustPush, server.PostBuild)
			repo.POST("/builds/:number/approve", session.MustPush, server.PostApproval)
			repo.POST("/builds/:number/decline", session.MustPush, server.PostDecline)
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

	ws := e.Group("/ws")
	{
		ws.GET("/feed", server.EventStream)
		ws.GET("/logs/:owner/:name/:build/:number",
			session.SetRepo(),
			session.SetPerm(),
			session.MustPull,
			server.LogStream,
		)
	}

	info := e.Group("/api/info")
	{
		info.GET("/queue",
			session.MustAdmin(),
			server.GetQueueInfo,
		)
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

	debugger := e.Group("/api/debug")
	{
		debugger.Use(session.MustAdmin())
		debugger.GET("/pprof/", debug.IndexHandler())
		debugger.GET("/pprof/heap", debug.HeapHandler())
		debugger.GET("/pprof/goroutine", debug.GoroutineHandler())
		debugger.GET("/pprof/block", debug.BlockHandler())
		debugger.GET("/pprof/threadcreate", debug.ThreadCreateHandler())
		debugger.GET("/pprof/cmdline", debug.CmdlineHandler())
		debugger.GET("/pprof/profile", debug.ProfileHandler())
		debugger.GET("/pprof/symbol", debug.SymbolHandler())
		debugger.POST("/pprof/symbol", debug.SymbolHandler())
		debugger.GET("/pprof/trace", debug.TraceHandler())
	}

	monitor := e.Group("/metrics")
	{
		monitor.GET("",
			session.MustAdmin(),
			metrics.PromHandler(),
		)
	}

	return e
}

// type FileHandler interface {
// 	Index(res http.ResponseWriter, data interface{}) error
// 	Login(res http.ResponseWriter, data interface{}) error
// 	Error(res http.ResponseWriter, data interface{}) error
// 	Asset(res http.ResponseWriter, req *http.Request)
// }
