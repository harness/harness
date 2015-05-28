package main

import (
	"flag"
	"html/template"
	"net/http"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/elazarl/go-bindata-assetfs"
	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/remote/github"
	"github.com/drone/drone/pkg/server"
	"github.com/drone/drone/pkg/server/session"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	eventbus "github.com/drone/drone/pkg/bus/builtin"
	queue "github.com/drone/drone/pkg/queue/builtin"
	runner "github.com/drone/drone/pkg/runner/builtin"
	store "github.com/drone/drone/pkg/store/builtin"

	_ "net/http/pprof"
)

var (
	// commit sha for the current build, set by
	// the compile process.
	version  string
	revision string
)

var (
	conf  = flag.String("config", "drone.toml", "")
	debug = flag.Bool("debug", false, "")
)

func main() {
	flag.Parse()

	settings, err := config.Load(*conf)
	if err != nil {
		panic(err)
	}

	db := store.MustConnect(settings.Database.Driver, settings.Database.Datasource)
	store := store.New(db)
	defer db.Close()

	remote := github.New(settings)
	session := session.New(settings)
	eventbus_ := eventbus.New()
	queue_ := queue.New()
	updater := runner.NewUpdater(eventbus_, store, remote)
	runner_ := runner.Runner{Updater: updater}

	// launch the local queue runner if the system
	// is not conifugred to run in agent mode
	if len(settings.Agents.Secret) != 0 {
		log.Infof("Run builds using remote build agents")
	} else {
		log.Infof("Run builds using the embedded build runner")
		go run(&runner_, queue_)
	}

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

	queue := api.Group("/queue")
	{
		queue.Use(server.MustAgent())
		queue.Use(server.SetSettings(settings))
		queue.Use(server.SetUpdater(updater))
		queue.POST("/pull", server.PollBuild)

		push := queue.Group("/push/:owner/:name")
		{
			push.Use(server.SetRepo())
			push.POST("", server.PushCommit)
			push.POST("/:commit", server.PushBuild)
			push.POST("/:commit/:build/logs", server.PushLogs)
		}
	}

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

	err = http.ListenAndServe(settings.Server.Addr, nil)
	if err != nil {
		log.Error("Cannot start server: ", err)
	}
}

// static is a helper function that will setup handlers
// for serving static files.
func static() http.Handler {
	// default file server is embedded
	var handler = http.FileServer(&assetfs.AssetFS{
		Asset:    Asset,
		AssetDir: AssetDir,
		Prefix:   "cmd/drone-server/static",
	})
	if *debug {
		handler = http.FileServer(
			http.Dir("cmd/drone-server/static"),
		)
	}
	return http.StripPrefix("/static/", handler)
}

// index is a helper function that will setup a template
// for rendering the main angular index.html file.
func index() *template.Template {
	file := MustAsset("cmd/drone-server/static/index.html")
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
