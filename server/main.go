package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/drone/config"
	"github.com/drone/drone/server/middleware"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/router"
	"github.com/drone/drone/shared/build/log"

	"github.com/GeertJohan/go.rice"

	"code.google.com/p/go.net/context"
	webcontext "github.com/goji/context"
	"github.com/zenazn/goji/web"

	_ "github.com/drone/drone/plugin/notify/email"
	"github.com/drone/drone/plugin/remote/bitbucket"
	"github.com/drone/drone/plugin/remote/github"
	"github.com/drone/drone/plugin/remote/gitlab"
	"github.com/drone/drone/plugin/remote/gogs"
	"github.com/drone/drone/server/blobstore"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/datastore/database"
	"github.com/drone/drone/server/worker/director"
	"github.com/drone/drone/server/worker/docker"
	"github.com/drone/drone/server/worker/pool"
)

const (
	DockerTLSWarning = `WARINING: Docker TLS cert or key not given, this may cause a build errors`
)

var (
	// commit sha for the current build, set by
	// the compile process.
	version  string
	revision string
)

var (
	// Database driver configuration. Defaults to sqlite
	// when no database configuration specified.
	datasource = config.String("database-datasource", "drone.sqlite")
	driver     = config.String("database-driver", "sqlite3")

	// HTTP Server settings.
	port   = config.String("server-port", ":8000")
	sslcrt = config.String("server-ssl-cert", "")
	sslkey = config.String("server-ssl-key", "")

	assets_folder = config.String("server-assets-folder", "")

	workers *pool.Pool
	worker  *director.Director
	pub     *pubsub.PubSub

	// Docker configuration details.
	dockercert = config.String("docker-cert", "")
	dockerkey  = config.String("docker-key", "")
	nodes      StringArr

	db *sql.DB
)

func main() {
	log.SetPriority(log.LOG_NOTICE)

	// Parses flags. The only flag that can be passed into the
	// application is the location of the configuration (.toml) file.
	var conf string
	flag.StringVar(&conf, "config", "", "")
	flag.Parse()

	config.Var(&nodes, "worker-nodes")

	// Parses config data. The config data can be stored in a config
	// file (.toml format) or environment variables, or a combo.
	config.SetPrefix("DRONE_")
	err := config.Parse(conf)
	if err != nil {
		log.Errf("Unable to parse config: %v", err)
		os.Exit(1)
	}

	// Setup the remote services. We need to execute these to register
	// the remote plugins with the system.
	//
	// NOTE: this cannot be done via init() because they need to be
	//       executed after config.Parse
	bitbucket.Register()
	github.Register()
	gitlab.Register()
	gogs.Register()

	// setup the database and cancel all pending
	// commits in the system.
	db = database.MustConnect(*driver, *datasource)
	go database.NewCommitstore(db).KillCommits()

	// Create the worker, director and builders
	workers = pool.New()
	worker = director.New()

	if nodes == nil || len(nodes) == 0 {
		workers.Allocate(docker.New())
		workers.Allocate(docker.New())
	} else {
		for _, node := range nodes {
			if strings.HasPrefix(node, "unix://") {
				workers.Allocate(docker.NewHost(node))
			} else if *dockercert != "" && *dockerkey != "" {
				workers.Allocate(docker.NewHostCertFile(node, *dockercert, *dockerkey))
			} else {
				fmt.Println(DockerTLSWarning)
				workers.Allocate(docker.NewHost(node))
			}
		}
	}

	pub = pubsub.NewPubSub()

	// create handler for static resources
	// if we have a configured assets folder it takes precedence over the assets
	// bundled to the binary
	var assetserve http.Handler
	if *assets_folder != "" {
		assetserve = http.FileServer(http.Dir(*assets_folder))
	} else {
		assetserve = http.FileServer(rice.MustFindBox("app").HTTPBox())
	}

	http.Handle("/robots.txt", assetserve)
	http.Handle("/static/", http.StripPrefix("/static", assetserve))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/"
		assetserve.ServeHTTP(w, r)
	})

	// create the router and add middleware
	mux := router.New()
	mux.Use(middleware.Options)
	mux.Use(ContextMiddleware)
	mux.Use(middleware.SetHeaders)
	mux.Use(middleware.SetUser)
	http.Handle("/api/", mux)

	// start the http server in either http or https mode,
	// depending on whether a certificate was provided.
	if len(*sslcrt) == 0 {
		panic(http.ListenAndServe(*port, nil))
	} else {
		panic(http.ListenAndServeTLS(*port, *sslcrt, *sslkey, nil))
	}
}

// ContextMiddleware creates a new go.net/context and
// injects into the current goji context.
func ContextMiddleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.Background()
		ctx = datastore.NewContext(ctx, database.NewDatastore(db))
		ctx = blobstore.NewContext(ctx, database.NewBlobstore(db))
		ctx = pool.NewContext(ctx, workers)
		ctx = director.NewContext(ctx, worker)
		ctx = pubsub.NewContext(ctx, pub)

		// add the context to the goji web context
		webcontext.Set(c, ctx)
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
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
