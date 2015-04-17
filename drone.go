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
		user.Use(server.SetSession(session))

		user.GET("", server.GetUserCurr)
		user.PATCH("", server.PutUserCurr)
		user.GET("/repos", server.GetUserRepos)
		user.GET("/tokens", server.GetUserTokens)
		user.POST("/tokens", server.PostToken)
		user.DELETE("/tokens/:label", server.DelToken)
	}

	users := api.Group("/users")
	{
		users.Use(server.MustAdmin())

		users.GET("", server.GetUsers)
		users.GET("/:name", server.GetUser)
		users.POST("/:name", server.PostUser)
		users.PATCH("/:name", server.PutUser)
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
			repo.PATCH("", server.PutRepo)
			repo.DELETE("", server.DeleteRepo)
			repo.POST("/watch", server.Subscribe)
			repo.DELETE("/unwatch", server.Unsubscribe)

			repo.GET("/builds", server.GetBuilds)
			repo.GET("/builds/:number", server.GetBuild)
			repo.POST("/builds/:number", server.RunBuild)
			repo.DELETE("/builds/:number", server.KillBuild)
			repo.GET("/logs/:number/:task", server.GetBuildLogs)
			repo.POST("/status/:number", server.PostBuildStatus)
		}
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
