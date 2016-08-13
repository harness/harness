package agent

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/drone/drone/client"
	"github.com/drone/drone/shared/token"
	"github.com/samalba/dockerclient"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"strings"
)

// AgentCmd is the exported command for starting the drone agent.
var AgentCmd = cli.Command{
	Name:   "agent",
	Usage:  "starts the drone agent",
	Action: start,
	Flags: []cli.Flag{
		cli.StringFlag{
			EnvVar: "DOCKER_HOST",
			Name:   "docker-host",
			Usage:  "docker deamon address",
			Value:  "unix:///var/run/docker.sock",
		},
		cli.BoolFlag{
			EnvVar: "DOCKER_TLS_VERIFY",
			Name:   "docker-tls-verify",
			Usage:  "docker daemon supports tlsverify",
		},
		cli.StringFlag{
			EnvVar: "DOCKER_CERT_PATH",
			Name:   "docker-cert-path",
			Usage:  "docker certificate directory",
			Value:  "",
		},
		cli.IntFlag{
			EnvVar: "DOCKER_MAX_PROCS",
			Name:   "docker-max-procs",
			Usage:  "limit number of running docker processes",
			Value:  2,
		},
		cli.StringFlag{
			EnvVar: "DOCKER_OS",
			Name:   "docker-os",
			Usage:  "docker operating system",
			Value:  "linux",
		},
		cli.StringFlag{
			EnvVar: "DOCKER_ARCH",
			Name:   "docker-arch",
			Usage:  "docker architecture system",
			Value:  "amd64",
		},
		cli.StringFlag{
			EnvVar: "DRONE_STORAGE_DRIVER",
			Name:   "drone-storage-driver",
			Usage:  "docker storage driver",
			Value:  "overlay",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER",
			Name:   "drone-server",
			Usage:  "drone server address",
			Value:  "http://localhost:8000",
		},
		cli.StringFlag{
			EnvVar: "DRONE_TOKEN",
			Name:   "drone-token",
			Usage:  "drone authorization token",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SECRET,DRONE_AGENT_SECRET",
			Name:   "drone-secret",
			Usage:  "drone agent secret",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_BACKOFF",
			Name:   "backoff",
			Usage:  "drone server backoff interval",
			Value:  time.Second * 15,
		},
		cli.DurationFlag{
			EnvVar: "DRONE_PING",
			Name:   "ping",
			Usage:  "drone server ping frequency",
			Value:  time.Minute * 5,
		},
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the agent in debug mode",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_TIMEOUT",
			Name:   "timeout",
			Usage:  "drone timeout due to log inactivity",
			Value:  time.Minute * 5,
		},
		cli.IntFlag{
			EnvVar: "DRONE_MAX_LOGS",
			Name:   "max-log-size",
			Usage:  "drone maximum log size in megabytes",
			Value:  5,
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_PRIVILEGED",
			Name:   "privileged",
			Usage:  "plugins that require privileged mode",
			Value: &cli.StringSlice{
				"plugins/docker",
				"plugins/docker:*",
				"plugins/gcr",
				"plugins/gcr:*",
				"plugins/ecr",
				"plugins/ecr:*",
			},
		},
		cli.StringFlag{
			EnvVar: "DRONE_PLUGIN_NAMESPACE",
			Name:   "namespace",
			Value:  "plugins",
			Usage:  "default plugin image namespace",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_PLUGIN_PULL",
			Name:   "pull",
			Usage:  "always pull latest plugin images",
		},
	},
}

func start(c *cli.Context) {

	// debug level if requested by user
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	var accessToken string
	if c.String("drone-secret") != "" {
		secretToken := c.String("drone-secret")
		accessToken, _ = token.New(token.AgentToken, "").Sign(secretToken)
	} else {
		accessToken = c.String("drone-token")
	}

	logrus.Infof("Connecting to %s with token %s",
		c.String("drone-server"),
		accessToken,
	)

	client := client.NewClientToken(
		strings.TrimRight(c.String("drone-server"),"/"),
		accessToken,
	)

	tls, err := dockerclient.TLSConfigFromCertPath(c.String("docker-cert-path"))
	if err == nil {
		tls.InsecureSkipVerify = c.Bool("docker-tls-verify")
	}
	docker, err := dockerclient.NewDockerClient(c.String("docker-host"), tls)
	if err != nil {
		logrus.Fatal(err)
	}

	go func() {
		for {
			if err := client.Ping(); err != nil {
				logrus.Warnf("unable to ping the server. %s", err.Error())
			}
			time.Sleep(c.Duration("ping"))
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < c.Int("docker-max-procs"); i++ {
		wg.Add(1)
		go func() {
			r := pipeline{
				drone:  client,
				docker: docker,
				config: config{
					platform:   c.String("docker-os") + "/" + c.String("docker-arch"),
					timeout:    c.Duration("timeout"),
					namespace:  c.String("namespace"),
					privileged: c.StringSlice("privileged"),
					pull:       c.BoolT("pull"),
					logs:       int64(c.Int("max-log-size")) * 1000000,
				},
			}
			for {
				if err := r.run(); err != nil {
					dur := c.Duration("backoff")
					logrus.Warnf("reconnect in %v. %s", dur, err.Error())
					time.Sleep(dur)
				}
			}
		}()
	}
	handleSignals()
	wg.Wait()
}

// tracks running builds
var running sync.WaitGroup

func handleSignals() {
	// Graceful shut-down on SIGINT/SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		logrus.Debugln("SIGTERM received.")
		logrus.Debugln("wait for running builds to finish.")
		running.Wait()
		logrus.Debugln("done.")
		os.Exit(0)
	}()
}
