package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/drone/config"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/database/connection"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/shared/build/log"
	"github.com/drone/drone/shared/model"

	"github.com/gorilla/pat"
	//"github.com/justinas/nosurf"
	"github.com/GeertJohan/go.rice"

	"github.com/drone/drone/plugin/remote/bitbucket"
	"github.com/drone/drone/plugin/remote/github"
	"github.com/drone/drone/plugin/remote/gitlab"
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
	version  string = "0.3-dev"
	revision string

	// Number of concurrent build workers to run
	// default to number of CPUs on machine
	workers int

	conf   string
	prefix string

	open bool

	nodes StringArr
)

func main() {
	log.SetPriority(log.LOG_NOTICE)

	flag.StringVar(&conf, "config", "", "")
	flag.StringVar(&prefix, "prefix", "DRONE_", "")
	flag.StringVar(&port, "port", ":8080", "")
	flag.StringVar(&sslcert, "sslcert", "", "")
	flag.StringVar(&sslkey, "sslkey", "", "")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "")
	flag.Parse()

	config.Var(&nodes, "worker-nodes")
	config.BoolVar(&open, "registration-open", false)
	config.StringVar(&driver, "database-driver", "sqlite3")
	config.StringVar(&datasource, "database-datasource", "drone.sqlite")

	config.SetPrefix(prefix)
	if err := config.Parse(conf); err != nil {
		fmt.Println("Error parsing config", err)
	}

	// setup the remote services
	bitbucket.Register()
	github.Register()
	gitlab.Register()

	// Create database connection
	conn := connection.NewConnection(driver, datasource)
	if err := conn.MigrateAll(); err != nil {
		panic(err)
	}
	defer conn.Close()

	// setup the database managers
	repos := database.NewRepoManager(conn.DB)
	users := database.NewUserManager(conn.DB)
	perms := database.NewPermManager(conn.DB)
	commits := database.NewCommitManager(conn.DB)
	servers := database.NewServerManager(conn.DB)
	remotes := database.NewRemoteManager(conn.DB)

	// message broker
	pubsub := pubsub.NewPubSub()

	// cancel all previously running builds
	go commits.CancelAll()

	queue := make(chan *model.Request)
	workerc := make(chan chan *model.Request)
	worker.NewDispatch(queue, workerc).Start()

	// if no worker nodes are specified than start 2 workers
	// using the default DOCKER_HOST
	if nodes == nil || len(nodes) == 0 {
		worker.NewWorker(workerc, users, repos, commits, pubsub, &model.Server{}).Start()
		worker.NewWorker(workerc, users, repos, commits, pubsub, &model.Server{}).Start()
	} else {
		for _, node := range nodes {
			println(node)
			worker.NewWorker(workerc, users, repos, commits, pubsub, &model.Server{Host: node}).Start()
		}
	}

	// setup the session managers
	sess := session.NewSession(users)

	// setup the router and register routes
	router := pat.New()
	handler.NewUsersHandler(users, sess).Register(router)
	handler.NewUserHandler(users, repos, commits, sess).Register(router)
	handler.NewHookHandler(users, repos, commits, remotes, queue).Register(router)
	handler.NewLoginHandler(users, repos, perms, sess, open).Register(router)
	handler.NewCommitHandler(users, repos, commits, perms, sess, queue).Register(router)
	handler.NewRepoHandler(repos, commits, perms, sess, remotes).Register(router)
	handler.NewBadgeHandler(repos, commits).Register(router)
	handler.NewServerHandler(servers, sess).Register(router)
	handler.NewRemoteHandler(users, remotes, sess).Register(router)
	handler.NewWsHandler(repos, commits, perms, sess, pubsub).Register(router)

	box := rice.MustFindBox("app/")
	fserver := http.FileServer(box.HTTPBox())
	index, _ := box.Bytes("index.html")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/favicon.ico"),
			strings.HasPrefix(r.URL.Path, "/scripts/"),
			strings.HasPrefix(r.URL.Path, "/styles/"),
			strings.HasPrefix(r.URL.Path, "/views/"):
			// serve static conent
			fserver.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/logout"),
			strings.HasPrefix(r.URL.Path, "/login/"),
			strings.HasPrefix(r.URL.Path, "/v1/"),
			strings.HasPrefix(r.URL.Path, "/ws/"):
			// standard header variables that should be set, for good measure.
			w.Header().Add("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
			w.Header().Add("X-Frame-Options", "DENY")
			w.Header().Add("X-Content-Type-Options", "nosniff")
			w.Header().Add("X-XSS-Protection", "1; mode=block")
			// serve dynamic content
			router.ServeHTTP(w, r)
		default:
			w.Write(index)
		}
	})

	// start webserver using HTTPS or HTTP
	if len(sslcert) != 0 {
		panic(http.ListenAndServeTLS(port, sslcert, sslkey, nil))
	} else {
		panic(http.ListenAndServe(port, nil))
	}
}

type StringArr []string

func (s *StringArr) String() string {
	return fmt.Sprint(*s)
}

func (s *StringArr) Set(value string) error {
	for _, str := range strings.Split(value, ",") {
		str = strings.TrimSpace(str)
		*s = append(*s, str)
	}
	return nil
}
