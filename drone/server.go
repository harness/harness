package main

import (
	"net/http"
	"time"

	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"
	"github.com/gin-gonic/contrib/ginrus"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var serverCmd = cli.Command{
	Name:  "server",
	Usage: "starts the drone server daemon",
	Action: func(c *cli.Context) {
		if err := server(c); err != nil {
			logrus.Fatal(err)
		}
	},
	Flags: []cli.Flag{
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the server in debug mode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER_ADDR",
			Name:   "server-addr",
			Usage:  "server address",
			Value:  ":8000",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER_CERT",
			Name:   "server-cert",
			Usage:  "server ssl cert",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER_KEY",
			Name:   "server-key",
			Usage:  "server ssl key",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_ADMIN",
			Name:   "admin",
			Usage:  "list of admin users",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_ORGS",
			Name:   "orgs",
			Usage:  "list of approved organizations",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_OPEN",
			Name:   "open",
			Usage:  "open user registration",
		},
		cli.StringFlag{
			EnvVar: "DRONE_YAML",
			Name:   "yaml",
			Usage:  "build configuraton file name",
			Value:  ".drone.yml",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_CACHE_TTY",
			Name:   "cache-tty",
			Usage:  "cache duration",
			Value:  time.Minute * 15,
		},
		cli.StringFlag{
			EnvVar: "DRONE_AGENT_SECRET,DRONE_SECRET",
			Name:   "agent-secret",
			Usage:  "agent secret passcode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_DATABASE_DRIVER,DATABASE_DRIVER",
			Name:   "driver",
			Usage:  "database driver",
			Value:  "sqlite3",
		},
		cli.StringFlag{
			EnvVar: "DRONE_DATABASE_DATASOURCE,DATABASE_CONFIG",
			Name:   "datasource",
			Usage:  "database driver configuration string",
			Value:  "drone.sqlite",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITHUB",
			Name:   "github",
			Usage:  "github driver is enabled",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_URL",
			Name:   "github-server",
			Usage:  "github server address",
			Value:  "https://github.com",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_CONTEXT",
			Name:   "github-context",
			Usage:  "github status context",
			Value:  "continuous-integration/drone",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_CLIENT",
			Name:   "github-client",
			Usage:  "github oauth2 client id",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_SECRET",
			Name:   "github-secret",
			Usage:  "github oauth2 client secret",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_GITHUB_SCOPE",
			Name:   "github-scope",
			Usage:  "github oauth scope",
			Value: &cli.StringSlice{
				"repo",
				"repo:status",
				"user:email",
				"read:org",
			},
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_GIT_USERNAME",
			Name:   "github-git-username",
			Usage:  "github machine user username",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_GIT_PASSWORD",
			Name:   "github-git-password",
			Usage:  "github machine user password",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_GITHUB_MERGE_REF",
			Name:   "github-merge-ref",
			Usage:  "github pull requests use merge ref",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITHUB_PRIVATE_MODE",
			Name:   "github-private-mode",
			Usage:  "github is running in private mode",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITHUB_SKIP_VERIFY",
			Name:   "github-skip-verify",
			Usage:  "github skip ssl verification",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GOGS",
			Name:   "gogs",
			Usage:  "gogs driver is enabled",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GOGS_URL",
			Name:   "gogs-server",
			Usage:  "gogs server address",
			Value:  "https://github.com",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GOGS_GIT_USERNAME",
			Name:   "gogs-git-username",
			Usage:  "gogs service account username",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GOGS_GIT_PASSWORD",
			Name:   "gogs-git-password",
			Usage:  "gogs service account password",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GOGS_PRIVATE_MODE",
			Name:   "gogs-private-mode",
			Usage:  "gogs private mode enabled",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GOGS_SKIP_VERIFY",
			Name:   "gogs-skip-verify",
			Usage:  "gogs skip ssl verification",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_BITBUCKET",
			Name:   "bitbucket",
			Usage:  "bitbucket driver is enabled",
		},
		cli.StringFlag{
			EnvVar: "DRONE_BITBUCKET_CLIENT",
			Name:   "bitbucket-client",
			Usage:  "bitbucket oauth2 client id",
		},
		cli.StringFlag{
			EnvVar: "DRONE_BITBUCKET_SECRET",
			Name:   "bitbucket-secret",
			Usage:  "bitbucket oauth2 client secret",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITLAB",
			Name:   "gitlab",
			Usage:  "gitlab driver is enabled",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITLAB_URL",
			Name:   "gitlab-server",
			Usage:  "gitlab server address",
			Value:  "https://gitlab.com",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITLAB_CLIENT",
			Name:   "gitlab-client",
			Usage:  "gitlab oauth2 client id",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITLAB_SECRET",
			Name:   "gitlab-secret",
			Usage:  "gitlab oauth2 client secret",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITLAB_GIT_USERNAME",
			Name:   "gitlab-git-username",
			Usage:  "gitlab service account username",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITLAB_GIT_PASSWORD",
			Name:   "gitlab-git-password",
			Usage:  "gitlab service account password",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITLAB_SKIP_VERIFY",
			Name:   "gitlab-skip-verify",
			Usage:  "gitlab skip ssl verification",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_GITLAB_PRIVATE_MODE",
			Name:   "gitlab-private-mode",
			Usage:  "gitlab is running in private mode",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_STASH",
			Name:   "stash",
			Usage:  "stash driver is enabled",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STASH_URL",
			Name:   "stash-server",
			Usage:  "stash server address",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STASH_CONSUMER_KEY",
			Name:   "stash-consumer-key",
			Usage:  "stash oauth1 consumer key",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STASH_CONSUMER_RSA",
			Name:   "stash-consumer-rsa",
			Usage:  "stash oauth1 private key file",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STASH_GIT_USERNAME",
			Name:   "stash-git-username",
			Usage:  "stash service account username",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STASH_GIT_PASSWORD",
			Name:   "stash-git-password",
			Usage:  "stash service account password",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_STASH_SKIP_VERIFY",
			Name:   "stash-skip-verify",
			Usage:  "stash skip ssl verification",
		},
	},
}

func server(c *cli.Context) error {

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
		middleware.Config(c),
		middleware.Queue(c),
		middleware.Stream(c),
		middleware.Bus(c),
		middleware.Cache(c),
		middleware.Store(c),
		middleware.Remote(c),
		middleware.Agents(c),
	)

	// start the server with tls enabled
	if c.String("server-cert") != "" {
		return http.ListenAndServeTLS(
			c.String("server-addr"),
			c.String("server-cert"),
			c.String("server-key"),
			handler,
		)
	}

	// start the server without tls enabled
	return http.ListenAndServe(
		c.String("server-addr"),
		handler,
	)
}
