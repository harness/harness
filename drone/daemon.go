package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"
	"github.com/gin-gonic/contrib/ginrus"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// DaemonCmd is the exported command for starting the drone server daemon.
var DaemonCmd = cli.Command{
	Name:  "daemon",
	Usage: "starts the drone server daemon",
	Action: func(c *cli.Context) {
		if err := start(c); err != nil {
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
			EnvVar: "DRONE_AGENT_SECRET",
			Name:   "agent-secret",
			Usage:  "agent secret passcode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_DATABASE_DRIVER,DATABASE_DRIVER",
			Name:   "driver",
			Usage:  "database driver",
			Value:  "sqite3",
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
			EnvVar: "DRONE_GITHUB_CLIENT",
			Name:   "github-client",
			Usage:  "github oauth2 client id",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GITHUB_SECRET",
			Name:   "github-sercret",
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
			Name:   "gitlab-sercret",
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
		//
		// remove these eventually
		//

		cli.BoolFlag{
			Name:   "agreement.ack",
			EnvVar: "I_UNDERSTAND_I_AM_USING_AN_UNSTABLE_VERSION",
			Usage:  "agree to terms of use.",
		},
		cli.BoolFlag{
			Name:   "agreement.fix",
			EnvVar: "I_AGREE_TO_FIX_BUGS_AND_NOT_FILE_BUGS",
			Usage:  "agree to terms of use.",
		},
	},
}

func start(c *cli.Context) error {

	if c.Bool("agreement.ack") == false || c.Bool("agreement.fix") == false {
		fmt.Println(agreement)
		os.Exit(1)
	}

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

//
// func setupCache(c *cli.Context) cache.Cache {
// 	return cache.NewTTL(
// 		c.Duration("cache-ttl"),
// 	)
// }
//
// func setupBus(c *cli.Context) bus.Bus {
// 	return bus.New()
// }
//
// func setupQueue(c *cli.Context) queue.Queue {
// 	return queue.New()
// }
//
// func setupStream(c *cli.Context) stream.Stream {
// 	return stream.New()
// }
//
// func setupStore(c *cli.Context) store.Store {
// 	return datastore.New(
// 		c.String("driver"),
// 		c.String("datasource"),
// 	)
// }
//
// func setupRemote(c *cli.Context) remote.Remote {
// 	var remote remote.Remote
// 	var err error
// 	switch {
// 	case c.Bool("github"):
// 		remote, err = setupGithub(c)
// 	case c.Bool("gitlab"):
// 		remote, err = setupGitlab(c)
// 	case c.Bool("bitbucket"):
// 		remote, err = setupBitbucket(c)
// 	case c.Bool("stash"):
// 		remote, err = setupStash(c)
// 	case c.Bool("gogs"):
// 		remote, err = setupGogs(c)
// 	default:
// 		err = fmt.Errorf("version control system not configured")
// 	}
// 	if err != nil {
// 		logrus.Fatalln(err)
// 	}
// 	return remote
// }
//
// func setupBitbucket(c *cli.Context) (remote.Remote, error) {
// 	return bitbucket.New(
// 		c.String("bitbucket-client"),
// 		c.String("bitbucket-server"),
// 	), nil
// }
//
// func setupGogs(c *cli.Context) (remote.Remote, error) {
// 	return gogs.New(gogs.Opts{
// 		URL:         c.String("gogs-server"),
// 		Username:    c.String("gogs-git-username"),
// 		Password:    c.String("gogs-git-password"),
// 		PrivateMode: c.Bool("gogs-private-mode"),
// 		SkipVerify:  c.Bool("gogs-skip-verify"),
// 	})
// }
//
// func setupStash(c *cli.Context) (remote.Remote, error) {
// 	return bitbucketserver.New(bitbucketserver.Opts{
// 		URL:         c.String("stash-server"),
// 		Username:    c.String("stash-git-username"),
// 		Password:    c.String("stash-git-password"),
// 		ConsumerKey: c.String("stash-consumer-key"),
// 		ConsumerRSA: c.String("stash-consumer-rsa"),
// 		SkipVerify:  c.Bool("stash-skip-verify"),
// 	})
// }
//
// func setupGitlab(c *cli.Context) (remote.Remote, error) {
// 	return gitlab.New(gitlab.Opts{
// 		URL:         c.String("gitlab-server"),
// 		Client:      c.String("gitlab-client"),
// 		Secret:      c.String("gitlab-sercret"),
// 		Username:    c.String("gitlab-git-username"),
// 		Password:    c.String("gitlab-git-password"),
// 		PrivateMode: c.Bool("gitlab-private-mode"),
// 		SkipVerify:  c.Bool("gitlab-skip-verify"),
// 	})
// }
//
// func setupGithub(c *cli.Context) (remote.Remote, error) {
// 	return github.New(
// 		c.String("github-server"),
// 		c.String("github-client"),
// 		c.String("github-sercret"),
// 		c.StringSlice("github-scope"),
// 		c.Bool("github-private-mode"),
// 		c.Bool("github-skip-verify"),
// 		c.BoolT("github-merge-ref"),
// 	)
// }
//
// func setupConfig(c *cli.Context) *server.Config {
// 	return &server.Config{
// 		Open:   c.Bool("open"),
// 		Yaml:   c.String("yaml"),
// 		Secret: c.String("agent-secret"),
// 		Admins: sliceToMap(c.StringSlice("admin")),
// 		Orgs:   sliceToMap(c.StringSlice("orgs")),
// 	}
// }
//
// func sliceToMap(s []string) map[string]bool {
// 	v := map[string]bool{}
// 	for _, ss := range s {
// 		v[ss] = true
// 	}
// 	return v
// }
//
// func printSecret(c *cli.Context) error {
// 	secret := c.String("agent-secret")
// 	if secret == "" {
// 		return fmt.Errorf("missing DRONE_AGENT_SECRET configuration parameter")
// 	}
// 	t := token.New(secret, "")
// 	s, err := t.Sign(secret)
// 	if err != nil {
// 		return fmt.Errorf("invalid value for DRONE_AGENT_SECRET. %s", s)
// 	}
//
// 	logrus.Infof("using agent secret %s", secret)
// 	logrus.Warnf("agents can connect with token %s", s)
// 	return nil
// }

var agreement = `
---


You are attempting to use the unstable channel. This build is experimental and
has known bugs and compatibility issues. It is not intended for general use.

Please consider using the latest stable release instead:

		drone/drone:0.4.2

If you are attempting to build from source please use the latest stable tag:

		v0.4.2

If you are interested in testing this experimental build AND assisting with
development you may proceed by setting the following environment:

		I_UNDERSTAND_I_AM_USING_AN_UNSTABLE_VERSION=true
		I_AGREE_TO_FIX_BUGS_AND_NOT_FILE_BUGS=true


---
`
