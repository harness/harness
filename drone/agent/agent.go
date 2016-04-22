package agent

import (
	"sync"
	"time"

	"github.com/drone/drone/client"
	"github.com/samalba/dockerclient"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
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
		cli.DurationFlag{
			EnvVar: "DRONE_BACKOFF",
			Name:   "backoff",
			Usage:  "drone server backoff interval",
			Value:  time.Second * 15,
		},
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the agent in debug mode",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_EXPERIMENTAL",
			Name:   "experimental",
			Usage:  "start the agent with experimental features",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_NETRC",
			Name:   "netrc-plugin",
			Usage:  "plugins that receive the netrc file",
			Value:  &cli.StringSlice{"git", "hg"},
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_PRIVILEGED",
			Name:   "privileged",
			Usage:  "plugins that require privileged mode",
			Value: &cli.StringSlice{
				"plugins/docker",
				"plugins/docker:*",
				"plguins/gcr",
				"plguins/gcr:*",
				"plugins/ecr",
				"plugins/ecr:*",
			},
		},
		cli.BoolFlag{
			EnvVar: "DRONE_PLUGIN_PULL",
			Name:   "pull",
			Usage:  "always pull latest plugin images",
		},
		cli.StringFlag{
			EnvVar: "DRONE_PLUGIN_NAMESPACE",
			Name:   "namespace",
			Value:  "plugins",
			Usage:  "default plugin image namespace",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_WHITELIST",
			Name:   "whitelist",
			Usage:  "plugins that are permitted to run on the host",
			Value:  &cli.StringSlice{"plugins/*"},
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

	client := client.NewClientToken(
		c.String("drone-server"),
		c.String("drone-token"),
	)

	tls, _ := dockerclient.TLSConfigFromCertPath(c.String("docker-cert-path"))
	if c.Bool("docker-host") {
		tls.InsecureSkipVerify = true
	}
	docker, err := dockerclient.NewDockerClient(c.String("docker-host"), tls)
	if err != nil {
		logrus.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < c.Int("docker-max-procs"); i++ {
		wg.Add(1)
		go func() {
			r := pipeline{
				drone:  client,
				docker: docker,
				config: config{
					whitelist:  c.StringSlice("whitelist"),
					namespace:  c.String("namespace"),
					privileged: c.StringSlice("privileged"),
					netrc:      c.StringSlice("netrc-plugin"),
					pull:       c.Bool("pull"),
				},
			}
			for {
				if err := r.run(); err != nil {
					dur := c.Duration("backoff")
					logrus.Warnf("Attempting to reconnect in %v", dur)
					time.Sleep(dur)
				}
			}
		}()
	}
	wg.Wait()
}
