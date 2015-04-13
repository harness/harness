package main

import (
	"flag"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/datastore/bolt"
	"github.com/drone/drone/remote/github"
	"github.com/drone/drone/server"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/settings"
)

var path = flag.String("config", "drone.toml", "")

func main() {
	flag.Parse()

	settings, err := settings.Parse(*path)
	if err != nil {
		panic(err)
	}
	remote := github.New(settings.Service)
	session := session.New(settings.Session)

	ds := bolt.Must(settings.Database.Path)
	defer ds.Close()

	r := gin.Default()

	api := r.Group("/api")
	api.Use(server.SetHeaders())
	api.Use(server.SetDatastore(ds))
	api.Use(server.SetRemote(remote))
	api.Use(server.SetSettings(settings))
	api.Use(server.SetUser(session))

	user := api.Group("/user")
	{
		user.Use(server.MustUser())

		user.GET("", server.GetUserCurr)
		user.PUT("", server.PutUserCurr)
		user.GET("/repos", server.GetUserRepos)
		user.GET("/tokens", server.GetUserTokens)
		user.POST("/tokens", server.PostToken)
		user.DELETE("/tokens", server.DelToken)
	}

	users := api.Group("/users")
	{
		users.Use(server.MustAdmin())

		users.GET("", server.GetUsers)
		users.GET("/:name", server.GetUser)
		users.PUT("/:name", server.PutUser)
		users.POST("/:name", server.PostUser)
		users.DELETE("/:name", server.DeleteUser)
	}

	repos := api.Group("/repos/:owner/:name")
	{
		repos.POST("", server.PostRepo)

		repo := repos.Group("")
		{
			repo.Use(server.SetRepo())
			repo.Use(server.SetPerm())
			repo.Use(server.CheckPull())
			repo.Use(server.CheckPush())

			repo.GET("", server.GetRepo)
			repo.PUT("", server.PutRepo)
			repo.DELETE("", server.DeleteRepo)
		}
	}

	subscribers := api.Group("/subscribers/:owner/:name")
	{
		subscribers.Use(server.SetRepo())
		subscribers.Use(server.SetPerm())
		subscribers.Use(server.CheckPull())

		subscribers.POST("", server.Subscribe)
		subscribers.DELETE("", server.Unsubscribe)
	}

	builds := api.Group("/builds/:owner/:name")
	{
		builds.Use(server.SetRepo())
		builds.Use(server.SetPerm())
		builds.Use(server.CheckPull())
		builds.Use(server.CheckPush())

		builds.GET("", server.GetBuilds)
		builds.GET("/:number", server.GetBuild)
		//TODO builds.POST("/:number", server.RestartBuild)
		//TODO builds.DELETE("/:number", server.CancelBuild)
	}

	tasks := api.Group("/tasks/:owner/:name/:number")
	{
		tasks.Use(server.SetRepo())
		tasks.Use(server.SetPerm())
		tasks.Use(server.CheckPull())
		tasks.Use(server.CheckPush())

		tasks.GET("", server.GetTasks)
		tasks.GET("/:task", server.GetTask)
	}

	logs := api.Group("/logs/:owner/:name/:number/:task")
	{
		logs.Use(server.SetRepo())
		logs.Use(server.SetPerm())
		logs.Use(server.CheckPull())
		logs.Use(server.CheckPush())

		logs.GET("", server.GetTaskLogs)
	}

	status := api.Group("/status/:owner/:name/:number")
	{
		status.Use(server.SetRepo())
		status.Use(server.SetPerm())
		status.Use(server.CheckPull())
		status.Use(server.CheckPush())

		status.GET("/:context", server.GetStatus)
		status.GET("", server.GetStatusList)
		status.POST("", server.PostStatus)
	}

	badges := api.Group("/badges/:owner/:name")
	{
		badges.Use(server.SetRepo())

		badges.GET("/status.svg", server.GetBadge)
		badges.GET("/cc.xml", server.GetCC)
	}

	hooks := api.Group("/hook")
	{
		hooks.POST("", server.PostHook)
	}

	auth := r.Group("/authorize")
	{
		auth.Use(server.SetHeaders())
		auth.Use(server.SetDatastore(ds))
		auth.Use(server.SetRemote(remote))
		auth.Use(server.SetSettings(settings))
		auth.Use(server.SetSession(session))
		auth.GET("", server.GetLogin)
		auth.POST("", server.GetLogin)
	}

	r.NoRoute(func(c *gin.Context) {
		c.File("server/static/index.html")
	})
	r.Static("/static", "server/static")
	r.Run(settings.Server.Addr)
}
