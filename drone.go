package main

import (
	"flag"

	"github.com/drone/drone/engine"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware/cache"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/shared/database"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/shared/server"

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
	database_ := database.Load(env)

	// setup the remote driver
	remote_ := remote.Load(env)

	// setup the runner
	engine_ := engine.Load(database_, env, remote_)

	// setup the server and start the listener
	server_ := server.Load(env)
	server_.Run(
		router.Load(
			header.Version(build),
			cache.Default(),
			context.SetDatabase(database_),
			context.SetRemote(remote_),
			context.SetEngine(engine_),
		),
	)
}
