package main

import (
	"net/http"
	"time"

	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
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
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	// setup the server and start the listener
	handler := router.Load(
		ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true),
		middleware.Version,
		middleware.Cache(),
		middleware.Store(),
		middleware.Remote(),
		middleware.Engine(),
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
