package main

import (
	"database/sql"
	"flag"
	"html/template"
	"net/http"
	"runtime"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/drone/drone/server/channel"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/queue"
	"github.com/drone/drone/server/render"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/config"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/build/docker"

	"github.com/gorilla/pat"
	//"github.com/justinas/nosurf"
	"github.com/GeertJohan/go.rice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

var (
	// port the server will run on
	port string

	// database driver used to connect to the database
	driver string

	// driver specific connection information. In this
	// case, it should be the location of the SQLite file
	datasource string

	// optional flags for tls listener
	sslcert string
	sslkey  string

	// commit sha for the current build.
	version  string = "0.2-dev"
	revision string

	// build will timeout after N milliseconds.
	// this will default to 500 minutes (6 hours)
	timeout time.Duration

	// Number of concurrent build workers to run
	// default to number of CPUs on machine
	workers int
)

// drone cofiguration data, loaded from the
// $HOME/.drone/config.toml file.
var conf config.Config

func main() {

	// parse command line flags
	flag.StringVar(&port, "port", ":8080", "")
	flag.StringVar(&driver, "driver", "sqlite3", "")
	flag.StringVar(&datasource, "datasource", "drone.sqlite", "")
	flag.StringVar(&sslcert, "sslcert", "", "")
	flag.StringVar(&sslkey, "sslkey", "", "")
	flag.DurationVar(&timeout, "timeout", 300*time.Minute, "")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "")
	flag.Parse()

	// parse the template files
	// TODO we need to retrieve these from go.rice
	//templ := template.Must(
	//	template.New("_").Funcs(render.FuncMap).ParseGlob("template/html/*.html"),
	//).ExecuteTemplate

	templateBox := rice.MustFindBox("template/html")
	templateFiles := []string{"login.html", "repo_branch.html", "repo_commit.html", "repo_conf.html", "repo_feed.html", "user_conf.html", "user_feed.html", "user_login.html", "user_repos.html", "404.html", "400.html"}
	templ := template.New("_").Funcs(render.FuncMap)
	for _, file := range templateFiles {
		templateData, _ := templateBox.String(file)
		templ, _ = templ.New(file).Parse(templateData)
	}

	// setup the database
	meddler.Default = meddler.SQLite
	db, _ := sql.Open(driver, datasource)
	database.Load(db)

	// setup the build queue
	queueRunner := queue.NewBuildRunner(docker.New(), timeout)
	queue := queue.Start(workers, queueRunner)

	// setup the database managers
	repos := repo.NewManager(db)
	users := user.NewManager(db)
	perms := perm.NewManager(db)
	commits := commit.NewManager(db)

	// cancel all previously running builds
	go commits.CancelAll()

	// setup the session managers
	sess := session.NewSession(users)

	// setup the router and register routes
	router := pat.New()
	handler.NewUsersHandler(users, sess).Register(router)
	handler.NewUserHandler(users, repos, commits, sess).Register(router)
	handler.NewHookHandler(users, repos, commits, &conf, queue).Register(router)
	handler.NewLoginHandler(users, repos, perms, sess, &conf).Register(router)
	handler.NewCommitHandler(repos, commits, perms, sess, queue).Register(router)
	handler.NewBranchHandler(repos, commits, perms, sess).Register(router)
	handler.NewRepoHandler(repos, commits, perms, sess, &conf).Register(router)
	handler.NewBadgeHandler(repos, commits).Register(router)
	handler.NewConfigHandler(conf, sess).Register(router)
	handler.NewSiteHandler(users, repos, commits, perms, sess, templ.ExecuteTemplate).Register(router)

	// serve static assets
	// TODO we need to replace this with go.rice
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(rice.MustFindBox("static/").HTTPBox())))

	// server websocket data
	http.Handle("/feed", websocket.Handler(channel.Read))

	// register the router
	// TODO we disabled nosurf because it was impacting API calls.
	//      we need to disable nosurf for api calls (ie not coming from website).
	http.Handle("/", router)

	// start webserver using HTTPS or HTTP
	if len(sslcert) != 0 && len(sslkey) != 0 {
		panic(http.ListenAndServeTLS(port, sslcert, sslkey, nil))
	} else {
		panic(http.ListenAndServe(port, nil))
	}
}
