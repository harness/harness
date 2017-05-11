package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"github.com/cncd/logging"
	"github.com/cncd/pubsub"
	"github.com/drone/drone/plugins/registry"
	"github.com/drone/drone/plugins/secrets"
	"github.com/drone/drone/plugins/sender"
	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"
	droneserver "github.com/drone/drone/server"
	"github.com/drone/drone/store"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/urfave/cli"
)

// Command exports the server command.
var Command = cli.Command{
	Name:   "server",
	Usage:  "starts the drone server daemon",
	Action: server,
	Flags: []cli.Flag{
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the server in debug mode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER_HOST,DRONE_HOST",
			Name:   "server-host",
			Usage:  "server host",
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
		cli.BoolFlag{
			EnvVar: "DRONE_LETS_ENCRYPT",
			Name:   "lets-encrypt",
			Usage:  "lets encrypt enabled",
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
		cli.DurationFlag{
			EnvVar: "DRONE_CACHE_TTL",
			Name:   "cache-ttl",
			Usage:  "cache duration",
			Value:  time.Minute * 15,
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_ESCALATE",
			Name:   "escalate",
			Value: &cli.StringSlice{
				"plugins/docker",
				"plugins/gcr",
				"plugins/ecr",
			},
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_VOLUME",
			Name:   "volume",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_NETWORK",
			Name:   "network",
		},
		cli.StringFlag{
			EnvVar: "DRONE_AGENT_SECRET,DRONE_SECRET",
			Name:   "agent-secret",
			Usage:  "agent secret passcode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SECRET_ENDPOINT",
			Name:   "secret-service",
			Usage:  "secret plugin endpoint",
		},
		cli.StringFlag{
			EnvVar: "DRONE_REGISTRY_ENDPOINT",
			Name:   "registry-service",
			Usage:  "registry plugin endpoint",
		},
		cli.StringFlag{
			EnvVar: "DRONE_GATEKEEPER_ENDPOINT",
			Name:   "gating-service",
			Usage:  "gated build endpoint",
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
			EnvVar: "DRONE_STASH_CONSUMER_RSA_STRING",
			Name:   "stash-consumer-rsa-string",
			Usage:  "stash oauth1 private key string",
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

	s := setupStore(c)
	setupEvilGlobals(c, s)

	// setup the server and start the listener
	handler := router.Load(
		ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true),
		middleware.Version,
		middleware.Config(c),
		middleware.Cache(c),
		middleware.Store(c, s),
		middleware.Remote(c),
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
	if !c.Bool("lets-encrypt") {
		return http.ListenAndServe(
			c.String("server-addr"),
			handler,
		)
	}

	// start the server with lets encrypt enabled
	// listen on ports 443 and 80
	var g errgroup.Group
	g.Go(func() error {
		return http.ListenAndServe(":http", handler)
	})

	g.Go(func() error {
		address, err := url.Parse(c.String("server-host"))
		if err != nil {
			return err
		}
		return http.Serve(autocert.NewListener(address.Host), handler)
	})

	return g.Wait()
}

// HACK please excuse the message during this period of heavy refactoring.
// We are currently transitioning from storing services (ie database, queue)
// in the gin.Context to storing them in a struct. We are also moving away
// from gin to gorilla. We will temporarily use global during our refactoring
// which will be removing in the final implementation.
func setupEvilGlobals(c *cli.Context, v store.Store) {

	// storage
	droneserver.Config.Storage.Files = v
	droneserver.Config.Storage.Config = v

	// services
	droneserver.Config.Services.Queue = setupQueue(c, v)
	droneserver.Config.Services.Logs = logging.New()
	droneserver.Config.Services.Pubsub = pubsub.New()
	droneserver.Config.Services.Pubsub.Create(context.Background(), "topic/events")
	droneserver.Config.Services.Registries = setupRegistryService(c, v)
	droneserver.Config.Services.Secrets = setupSecretService(c, v)
	droneserver.Config.Services.Senders = sender.New(v, v)
	if endpoint := c.String("registry-service"); endpoint != "" {
		droneserver.Config.Services.Registries = registry.NewRemote(endpoint)
	}
	if endpoint := c.String("secret-service"); endpoint != "" {
		droneserver.Config.Services.Secrets = secrets.NewRemote(endpoint)
	}
	if endpoint := c.String("gating-service"); endpoint != "" {
		droneserver.Config.Services.Senders = sender.NewRemote(endpoint)
	}

	// server configuration
	droneserver.Config.Server.Cert = c.String("server-cert")
	droneserver.Config.Server.Key = c.String("server-key")
	droneserver.Config.Server.Pass = c.String("agent-secret")
	droneserver.Config.Server.Host = c.String("server-host")
	droneserver.Config.Server.Port = c.String("server-addr")
	droneserver.Config.Pipeline.Networks = c.StringSlice("network")
	droneserver.Config.Pipeline.Volumes = c.StringSlice("volume")
	droneserver.Config.Pipeline.Privileged = c.StringSlice("escalate")
	// droneserver.Config.Server.Open = cli.Bool("open")
	// droneserver.Config.Server.Orgs = sliceToMap(cli.StringSlice("orgs"))
	// droneserver.Config.Server.Admins = sliceToMap(cli.StringSlice("admin"))
}
