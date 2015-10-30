package main

import (
	"flag"

	"github.com/CiscoCloud/drone/engine"
	"github.com/CiscoCloud/drone/remote"
	"github.com/CiscoCloud/drone/router"
	"github.com/CiscoCloud/drone/router/middleware/cache"
	"github.com/CiscoCloud/drone/router/middleware/context"
	"github.com/CiscoCloud/drone/router/middleware/header"
	"github.com/CiscoCloud/drone/shared/envconfig"
	"github.com/CiscoCloud/drone/shared/server"
	"github.com/CiscoCloud/drone/store/datastore"

	"github.com/Sirupsen/logrus"
)

// build revision number populated by the continuous
// integration server at compile time.
var build string = "custom"

var (
	dotenv = flag.String("config", ".env", "")
	debug  = flag.Bool("debug", false, "")
)

func main() {
	flag.Parse()

	// debug level if requested by user
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Load the configuration from env file
	env := envconfig.Load(*dotenv)

	// Setup the database driver
	store_ := datastore.Load(env)

	// setup the remote driver
	remote_ := remote.Load(env)

	// setup the runner
	engine_ := engine.Load(env, store_)

	// setup the server and start the listener
	server_ := server.Load(env)
	server_.Run(
		router.Load(
			header.Version(build),
			cache.Default(),
			context.SetStore(store_),
			context.SetRemote(remote_),
			context.SetEngine(engine_),
		),
	)
}
