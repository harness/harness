package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"github.com/cncd/logging"
	"github.com/cncd/pipeline/pipeline/rpc/proto"
	"github.com/cncd/pubsub"
	"github.com/drone/drone/plugins/registry"
	"github.com/drone/drone/plugins/secrets"
	"github.com/drone/drone/plugins/sender"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router"
	"github.com/drone/drone/router/middleware"
	droneserver "github.com/drone/drone/server"
	"github.com/drone/drone/store"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/urfave/cli"
	oldcontext "golang.org/x/net/context"
)

var flags = []cli.Flag{
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
	//
	// resource limit parameters
	//
	cli.Int64Flag{
		Name:   "limit-mem-swap",
		EnvVar: "DRONE_LIMIT_MEM_SWAP",
	},
	cli.Int64Flag{
		Name:   "limit-mem",
		EnvVar: "DRONE_LIMIT_MEM",
	},
	cli.Int64Flag{
		Name:   "limit-shm-size",
		EnvVar: "DRONE_LIMIT_SHM_SIZE",
	},
	cli.Int64Flag{
		Name:   "limit-cpu-quota",
		EnvVar: "DRONE_LIMIT_CPU_QUOTA",
	},
	cli.Int64Flag{
		Name:   "limit-cpu-shares",
		EnvVar: "DRONE_LIMIT_CPU_SHARES",
	},
	cli.StringFlag{
		Name:   "limit-cpu-set",
		EnvVar: "DRONE_LIMIT_CPU_SET",
	},
	//
	// remote parameters
	//
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
		EnvVar: "DRONE_GITEA",
		Name:   "gitea",
		Usage:  "gitea driver is enabled",
	},
	cli.StringFlag{
		EnvVar: "DRONE_GITEA_URL",
		Name:   "gitea-server",
		Usage:  "gitea server address",
		Value:  "https://try.gitea.io",
	},
	cli.StringFlag{
		EnvVar: "DRONE_GITEA_GIT_USERNAME",
		Name:   "gitea-git-username",
		Usage:  "gitea service account username",
	},
	cli.StringFlag{
		EnvVar: "DRONE_GITEA_GIT_PASSWORD",
		Name:   "gitea-git-password",
		Usage:  "gitea service account password",
	},
	cli.BoolFlag{
		EnvVar: "DRONE_GITEA_PRIVATE_MODE",
		Name:   "gitea-private-mode",
		Usage:  "gitea private mode enabled",
	},
	cli.BoolFlag{
		EnvVar: "DRONE_GITEA_SKIP_VERIFY",
		Name:   "gitea-skip-verify",
		Usage:  "gitea skip ssl verification",
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
}

func server(c *cli.Context) error {

	// debug level if requested by user
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	// must configure the drone_host variable
	if c.String("server-host") == "" {
		logrus.Fatalln("DRONE_HOST is not properly configured")
	}

	remote_, err := SetupRemote(c)
	if err != nil {
		logrus.Fatal(err)
	}

	store_ := setupStore(c)
	setupEvilGlobals(c, store_, remote_)

	// setup the server and start the listener
	handler := router.Load(
		ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true),
		middleware.Version,
		middleware.Config(c),
		middleware.Cache(c),
		middleware.Store(c, store_),
		middleware.Remote(remote_),
	)

	var g errgroup.Group

	// start the grpc server
	g.Go(func() error {

		lis, err := net.Listen("tcp", ":9000")
		if err != nil {
			logrus.Error(err)
			return err
		}
		auther := &authorizer{
			password: c.String("agent-secret"),
		}
		s := grpc.NewServer(
			grpc.StreamInterceptor(auther.streamInterceptor),
			grpc.UnaryInterceptor(auther.unaryIntercaptor),
		)
		ss := new(droneserver.DroneServer)
		ss.Queue = droneserver.Config.Services.Queue
		ss.Logger = droneserver.Config.Services.Logs
		ss.Pubsub = droneserver.Config.Services.Pubsub
		ss.Remote = remote_
		ss.Store = store_
		ss.Host = droneserver.Config.Server.Host
		proto.RegisterDroneServer(s, ss)

		err = s.Serve(lis)
		if err != nil {
			logrus.Error(err)
			return err
		}
		return nil
	})

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
func setupEvilGlobals(c *cli.Context, v store.Store, r remote.Remote) {

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
	droneserver.Config.Services.Environ = setupEnvironService(c, v)

	if endpoint := c.String("registry-service"); endpoint != "" {
		droneserver.Config.Services.Registries = registry.NewRemote(endpoint)
	}
	if endpoint := c.String("secret-service"); endpoint != "" {
		droneserver.Config.Services.Secrets = secrets.NewRemote(endpoint)
	}
	if endpoint := c.String("gating-service"); endpoint != "" {
		droneserver.Config.Services.Senders = sender.NewRemote(endpoint)
	}

	// limits
	droneserver.Config.Pipeline.Limits.MemSwapLimit = c.Int64("limit-mem-swap")
	droneserver.Config.Pipeline.Limits.MemLimit = c.Int64("limit-mem")
	droneserver.Config.Pipeline.Limits.ShmSize = c.Int64("limit-shm-size")
	droneserver.Config.Pipeline.Limits.CPUQuota = c.Int64("limit-cpu-quota")
	droneserver.Config.Pipeline.Limits.CPUShares = c.Int64("limit-cpu-shares")
	droneserver.Config.Pipeline.Limits.CPUSet = c.String("limit-cpu-set")

	// server configuration
	droneserver.Config.Server.Cert = c.String("server-cert")
	droneserver.Config.Server.Key = c.String("server-key")
	droneserver.Config.Server.Pass = c.String("agent-secret")
	droneserver.Config.Server.Host = strings.TrimRight(c.String("server-host"), "/")
	droneserver.Config.Server.Port = c.String("server-addr")
	droneserver.Config.Pipeline.Networks = c.StringSlice("network")
	droneserver.Config.Pipeline.Volumes = c.StringSlice("volume")
	droneserver.Config.Pipeline.Privileged = c.StringSlice("escalate")
	// droneserver.Config.Server.Open = cli.Bool("open")
	// droneserver.Config.Server.Orgs = sliceToMap(cli.StringSlice("orgs"))
	// droneserver.Config.Server.Admins = sliceToMap(cli.StringSlice("admin"))
}

type authorizer struct {
	username string
	password string
}

func (a *authorizer) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := a.authorize(stream.Context()); err != nil {
		return err
	}
	return handler(srv, stream)
}

func (a *authorizer) unaryIntercaptor(ctx oldcontext.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	if err := a.authorize(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (a *authorizer) authorize(ctx context.Context) error {
	if md, ok := metadata.FromContext(ctx); ok {
		if len(md["password"]) > 0 && md["password"][0] == a.password {
			return nil
		}
		return errors.New("invalid agent token")
	}
	return errors.New("missing agent token")
}
