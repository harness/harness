package main

import (
	"database/sql"
	"flag"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/drone/drone/server/channel"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/database/schema"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/queue"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/build/docker"
	"github.com/drone/drone/shared/build/log"

	"github.com/gorilla/pat"
	//"github.com/justinas/nosurf"
	"github.com/GeertJohan/go.rice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

var (
	// home directory for the application.
	home string

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

func main() {

	log.SetPriority(log.LOG_NOTICE)

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

	// load the html templates
	templateBox := rice.MustFindBox("template/html")
	templateFiles := []string{"login.html", "repo_branch.html", "repo_commit.html", "repo_conf.html", "repo_feed.html", "user_conf.html", "user_feed.html", "user_login.html", "user_repos.html", "404.html", "400.html"}
	templ := template.New("_").Funcs(funcMap)
	for _, file := range templateFiles {
		templateData, _ := templateBox.String(file)
		templ, _ = templ.New(file).Parse(templateData)
	}

	// setup the database
	meddler.Default = meddler.SQLite
	db, _ := sql.Open(driver, datasource)
	schema.Load(db)

	// setup the database managers
	repos := database.NewRepoManager(db)
	users := database.NewUserManager(db)
	perms := database.NewPermManager(db)
	commits := database.NewCommitManager(db)
	configs := database.NewConfigManager(filepath.Join(home, "config.toml"))

	// cancel all previously running builds
	go commits.CancelAll()

	// setup the build queue
	queueRunner := queue.NewBuildRunner(docker.New(), timeout)
	queue := queue.Start(workers, commits, queueRunner)

	// setup the session managers
	sess := session.NewSession(users)

	// setup the router and register routes
	router := pat.New()
	handler.NewUsersHandler(users, sess).Register(router)
	handler.NewUserHandler(users, repos, commits, sess).Register(router)
	handler.NewHookHandler(users, repos, commits, configs, queue).Register(router)
	handler.NewLoginHandler(users, repos, perms, sess, configs).Register(router)
	handler.NewCommitHandler(repos, commits, perms, sess, queue).Register(router)
	handler.NewBranchHandler(repos, commits, perms, sess).Register(router)
	handler.NewRepoHandler(repos, commits, perms, sess, configs).Register(router)
	handler.NewBadgeHandler(repos, commits).Register(router)
	handler.NewConfigHandler(configs, sess).Register(router)
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

// initialize the .drone directory and create a skeleton config
// file if one does not already exist.
func init() {
	// load the current user
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	// set .drone home dir
	home = filepath.Join(u.HomeDir, ".drone")

	// create the .drone home directory
	os.MkdirAll(home, 0777)

	// check for the config file
	filename := filepath.Join(u.HomeDir, ".drone", "config.toml")
	if _, err := os.Stat(filename); err != nil {
		// if not exists, create
		ioutil.WriteFile(filename, []byte(defaultConfig), 0777)
	}
}

var defaultConfig = `
# Enables user self-registration. If false, the system administrator
# will need to manually add users to the system.
registration = true

[smtp]
host = ""
port = ""
from = ""
username = ""
password = ""

[bitbucket]
url = "https://bitbucket.org"
api = "https://bitbucket.org"
client = ""
secret = ""
enabled = false

[github]
url = "https://github.com"
api = "https://api.github.com"
client = ""
secret = ""
enabled = false

[githubenterprise]
url = ""
api = ""
client = ""
secret = ""
enabled = false

[gitlab]
url = ""
api = ""
client = ""
secret = ""
enabled = false

[stash]
url = ""
api = ""
client = ""
secret = ""
enabled = false
`
