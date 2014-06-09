package main

import (
	"database/sql"
	"flag"
	"html/template"
	"net/http"

	"code.google.com/p/go.net/websocket"
	"github.com/drone/drone/server/channel"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/render"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/config"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
	"github.com/drone/drone/server/session"

	"github.com/gorilla/pat"
	//"github.com/justinas/nosurf"
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
	flag.Parse()

	// parse the template files
	// TODO we need to retrieve these from go.rice
	templ := template.Must(
		template.New("_").Funcs(render.FuncMap).ParseGlob("template/html/*.html"),
	).ExecuteTemplate

	// setup the database
	meddler.Default = meddler.SQLite
	db, _ := sql.Open(driver, datasource)
	database.Load(db)

	// setup the database managers
	repos := repo.NewManager(db)
	users := user.NewManager(db)
	perms := perm.NewManager(db)
	commits := commit.NewManager(db)

	// setup the session managers
	sess := session.NewSession(users)

	// setup the router and register routes
	router := pat.New()
	handler.NewUsersHandler(users, sess).Register(router)
	handler.NewUserHandler(users, repos, commits, sess).Register(router)
	handler.NewHookHandler(users, repos, commits, &conf).Register(router)
	handler.NewLoginHandler(users, repos, perms, sess, &conf).Register(router)
	handler.NewCommitHandler(repos, commits, perms, sess).Register(router)
	handler.NewBranchHandler(repos, commits, perms, sess).Register(router)
	handler.NewRepoHandler(repos, commits, perms, sess, &conf).Register(router)
	handler.NewBadgeHandler(repos, commits).Register(router)
	handler.NewConfigHandler(conf, sess).Register(router)
	handler.NewSiteHandler(users, repos, commits, perms, sess, templ).Register(router)

	// serve static assets
	// TODO we need to replace this with go.rice
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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
