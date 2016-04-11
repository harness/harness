package main

import (
	"net/http"

	"github.com/drone/drone/engine"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware/cache"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/store/datastore"

	"github.com/Sirupsen/logrus"
	"github.com/ianschenck/envflag"
	_ "github.com/joho/godotenv/autoload"
)

var (
	addr = envflag.String("SERVER_ADDR", ":8000", "")
	cert = envflag.String("SERVER_CERT", "", "")
	key  = envflag.String("SERVER_KEY", "", "")

	debug = envflag.Bool("DEBUG", false, "")
)

func main() {
	envflag.Parse()

	// debug level if requested by user
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Load the configuration from env file
	env := envconfig.Load(".env")

	// Setup the database driver
	store_ := datastore.Load(env)

	// setup the remote driver
	remote_ := remote.Load(env)

	// setup the runner
	engine_ := engine.Load(env, store_)

	// setup the server and start the listener
	handler := router.Load(
		header.Version,
		cache.Default(),
		context.SetStore(store_),
		context.SetRemote(remote_),
		context.SetEngine(engine_),
	)

	if *cert != "" {
		logrus.Fatal(
			http.ListenAndServeTLS(*addr, *cert, *key, handler),
		)
	} else {
		logrus.Fatal(
			http.ListenAndServe(*addr, handler),
		)
	}
}
