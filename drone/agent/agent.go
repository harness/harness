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
			EnvVar: "BACKOFF",
			Name:   "drone-backoff",
			Usage:  "drone server backoff interval",
			Value:  time.Second * 15,
		},
		cli.BoolFlag{
			EnvVar: "DEBUG",
			Name:   "debug",
			Usage:  "start the agent in debug mode",
		},
		cli.BoolFlag{
			EnvVar: "EXPERIMENTAL",
			Name:   "experimental",
			Usage:  "start the agent with experimental features",
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
			for {
				if err := recoverExec(client, docker); err != nil {
					dur := c.Duration("drone-backoff")
					logrus.Debugf("Attempting to reconnect in %v", dur)
					time.Sleep(dur)
				}
			}
		}()
	}
	wg.Wait()
}
