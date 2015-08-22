package main

import (
	"html/template"
	"net/http"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/namsral/flag"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/elazarl/go-bindata-assetfs"
	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/remote"
	"github.com/drone/drone/pkg/server"
	"github.com/drone/drone/pkg/server/session"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	eventbus "github.com/drone/drone/pkg/bus/builtin"
	queue "github.com/drone/drone/pkg/queue/builtin"
	runner "github.com/drone/drone/pkg/runner/builtin"
	"github.com/drone/drone/pkg/store"

	_ "github.com/drone/drone/pkg/remote/builtin/github"
	_ "github.com/drone/drone/pkg/remote/builtin/gitlab"
	_ "github.com/drone/drone/pkg/store/builtin"

	_ "net/http/pprof"
)

var (
	// commit sha for the current build, set by
	// the compile process.
	version  string
	revision string
)

var conf = struct {
	debug bool

	server struct {
		addr string
		cert string
		key  string
	}

	session struct {
		expiry string
		secret string
	}

	docker struct {
		host string
		cert string
		key  string
		ca   string
	}

	remote struct {
		driver string
		config string
	}

	database struct {
		driver string
		config string
	}

	plugin struct {
		filter string
	}
}{}

func main() {

	flag.StringVar(&conf.docker.host, "docker-host", "unix:///var/run/docker.sock", "")
	flag.StringVar(&conf.docker.cert, "docker-cert", "", "")
	flag.StringVar(&conf.docker.key, "docker-key", "", "")
	flag.StringVar(&conf.docker.ca, "docker-ca", "", "")
	flag.StringVar(&conf.server.addr, "server-addr", ":8080", "")
	flag.StringVar(&conf.server.cert, "server-cert", "", "")
	flag.StringVar(&conf.server.key, "server-key", "", "")
	flag.StringVar(&conf.session.expiry, "session-expiry", "", "")
	flag.StringVar(&conf.session.secret, "session-secret", "", "")
	flag.StringVar(&conf.remote.driver, "remote-driver", "github", "")
	flag.StringVar(&conf.remote.config, "remote-config", "https://github.com", "")
	flag.StringVar(&conf.database.driver, "database-driver", "sqlite3", "")
	flag.StringVar(&conf.database.config, "database-config", "drone.sqlite", "")
	flag.StringVar(&conf.plugin.filter, "plugin-filter", "plugins/*", "")
	flag.BoolVar(&conf.debug, "debug", false, "")

	flag.String("config", "", "")
	flag.Parse()

	settings, err := config.Load()
	if err != nil {
		panic(err)
	}
	settings.Plugins = []string{conf.plugin.filter} // todo hack

	store, err := store.New(conf.database.driver, conf.database.config)
	if err != nil {
		panic(err)
	}

	remote, err := remote.New(conf.remote.driver, conf.remote.config)
	if err != nil {
		panic(err)
	}

	session := session.New(conf.remote.config)
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
	api.OPTIONS("/*path", func(c *gin.Context) {})

	user := api.Group("/user")
	{
		user.Use(server.MustUser())

		user.GET("", server.GetUserCurr)
		user.PATCH("", server.PutUserCurr)
		user.GET("/feed", server.GetUserFeed)
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
			repo.POST("/encrypt", server.Encrypt)
			repo.POST("/watch", server.Subscribe)
			repo.DELETE("/unwatch", server.Unsubscribe)

			repo.GET("/builds", server.GetCommits)
			repo.GET("/builds/:number", server.GetCommit)
			repo.POST("/builds/:number", server.RunBuild)
			repo.DELETE("/builds/:number", server.KillBuild)
			repo.GET("/logs/:number/:task", server.GetLogs)
			// repo.POST("/status/:number", server.PostBuildStatus)
		}

		// Routes for external services
		repoExternal := repos.Group("")
		{
			repoExternal.Use(server.SetRepo())

			repoExternal.GET("/pr/:number", server.GetPullRequest)
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

	redirects := r.Group("/redirect")
	{
		redirects.Use(server.SetDatastore(store))
		redirects.Use(server.SetRepo())

		redirects.GET("/:owner/:name/commit/:sha", server.RedirectSha)
	}

	r.SetHTMLTemplate(index())
	r.NoRoute(func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	http.Handle("/static/", static())
	http.Handle("/", r)

	err = http.ListenAndServe(conf.server.addr, nil)
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
	if conf.debug {
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
