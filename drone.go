package main

import (
	"flag"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/remote/github"
	"github.com/drone/drone/server"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/settings"
	"github.com/elazarl/go-bindata-assetfs"

	store "github.com/drone/drone/datastore/builtin"
	eventbus "github.com/drone/drone/eventbus/builtin"
	queue "github.com/drone/drone/queue/builtin"
	runner "github.com/drone/drone/runner/builtin"

	_ "net/http/pprof"
)

var conf = flag.String("config", "drone.toml", "")

func main() {
	flag.Parse()

	settings, err := settings.Parse(*conf)
	if err != nil {
		panic(err)
	}

	db := store.MustConnect(settings.Database.Driver, settings.Database.Datasource)
	store := store.New(db)
	defer db.Close()

	remote := github.New(settings.Service)
	session := session.New(settings.Session)
	eventbus_ := eventbus.New()
	queue_ := queue.New()
	updater := runner.NewUpdater(eventbus_, store, remote)
	runner_ := runner.Runner{Updater: updater}
	go run(&runner_, queue_)

	r := gin.Default()

	api := r.Group("/api")
	api.Use(server.SetHeaders())
	api.Use(server.SetBus(eventbus_))
	api.Use(server.SetDatastore(store))
	api.Use(server.SetRemote(remote))
	api.Use(server.SetQueue(queue_))
	api.Use(server.SetSettings(settings))
	api.Use(server.SetSession(session))
	api.Use(server.SetUser(session))
	api.Use(server.SetRunner(&runner_))

	user := api.Group("/user")
	{
		user.Use(server.MustUser())

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

	agents := api.Group("/agents")
	{
		agents.Use(server.MustAdmin())
		agents.GET("/token", server.GetAgentToken)
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

			repo.GET("/builds", server.GetCommits)
			repo.GET("/builds/:number", server.GetCommit)
			repo.POST("/builds/:number", server.RunBuild)
			repo.DELETE("/builds/:number", server.KillBuild)
			repo.GET("/logs/:number/:task", server.GetLogs)
			// repo.POST("/status/:number", server.PostBuildStatus)
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

	// queue := api.Group("/queue")
	// {
	// 	queue.Use(server.MustAgent())
	// 	queue.GET("", server.GetQueue)
	// 	queue.POST("/pull", server.PollBuild)

	// 	push := queue.Group("/push/:owner/:name")
	// 	{
	// 		push.Use(server.SetRepo())
	// 		push.POST("", server.PushBuild)
	// 		push.POST("/:build", server.PushTask)
	// 		push.POST("/:build/:task/logs", server.PushLogs)
	// 	}
	// }

	stream := api.Group("/stream")
	{
		stream.Use(server.SetRepo())
		stream.Use(server.SetPerm())
		stream.GET("/:owner/:name", server.GetRepoEvents)
		stream.GET("/:owner/:name/:build/:number", server.GetStream)

	}

	auth := r.Group("/authorize")
	{
		auth.Use(server.SetHeaders())
		auth.Use(server.SetDatastore(store))
		auth.Use(server.SetRemote(remote))
		auth.Use(server.SetSettings(settings))
		auth.Use(server.SetSession(session))
		auth.GET("", server.GetLogin)
		auth.POST("", server.GetLogin)
	}

	r.SetHTMLTemplate(index())
	r.NoRoute(func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	http.Handle("/static/", static())
	http.Handle("/", r)
	http.ListenAndServe(settings.Server.Addr, nil)
}

// static is a helper function that will setup handlers
// for serving static files.
func static() http.Handler {
	return http.StripPrefix("/static/", http.FileServer(&assetfs.AssetFS{
		Asset:    Asset,
		AssetDir: AssetDir,
		Prefix:   "server/static",
	}))
}

// index is a helper function that will setup a template
// for rendering the main angular index.html file.
func index() *template.Template {
	file := MustAsset("server/static/index.html")
	filestr := string(file)
	return template.Must(template.New("index.html").Parse(filestr))
}

// run is a helper function for initializing the
// built-in build runner, if not running in remote
// mode.
func run(r *runner.Runner, q *queue.Queue) {
	defer func() {
		recover()
	}()
	r.Poll(q)
}
