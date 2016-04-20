package server

import (
	"net/http"
	"time"

	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gin-gonic/contrib/ginrus"
)

// ServeCmd is the exported command for starting the drone server.
var ServeCmd = cli.Command{
	Name:  "serve",
	Usage: "starts the drone server",
	Action: func(c *cli.Context) {
		if err := start(c); err != nil {
			logrus.Fatal(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			EnvVar: "SERVER_ADDR",
			Name:   "server-addr",
			Usage:  "server address",
			Value:  ":8000",
		},
		cli.StringFlag{
			EnvVar: "SERVER_CERT",
			Name:   "server-cert",
			Usage:  "server ssl cert",
		},
		cli.StringFlag{
			EnvVar: "SERVER_KEY",
			Name:   "server-key",
			Usage:  "server ssl key",
		},
		cli.BoolFlag{
			EnvVar: "DEBUG",
			Name:   "debug",
			Usage:  "start the server in debug mode",
		},
		cli.BoolFlag{
			EnvVar: "EXPERIMENTAL",
			Name:   "experimental",
			Usage:  "start the server with experimental features",
		},
	},
}

func start(c *cli.Context) error {

	// debug level if requested by user
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	// setup the server and start the listener
	handler := router.Load(
		ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true),
		middleware.Version,
		middleware.Queue(),
		middleware.Stream(),
		middleware.Bus(),
		middleware.Cache(),
		middleware.Store(),
		middleware.Remote(),
		middleware.Engine(),
	)

	if c.String("server-cert") != "" {
		return http.ListenAndServeTLS(
			c.String("server-addr"),
			c.String("server-cert"),
			c.String("server-key"),
			handler,
		)
	}

	return http.ListenAndServe(
		c.String("server-addr"),
		handler,
	)
}
