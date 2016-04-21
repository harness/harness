package docker

import (
	"os"

	"github.com/drone/drone/engine/runner"
	"github.com/samalba/dockerclient"
)

var (
	dockerHost = os.Getenv("DOCKER_HOST")
	dockerCert = os.Getenv("DOCKER_CERT_PATH")
	dockerTLS  = os.Getenv("DOCKER_TLS_VERIFY")
)

func init() {
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}
}

// New returns a new Docker engine using the provided Docker client.
func New(client dockerclient.Client) runner.Engine {
	return &dockerEngine{client}
}

// NewEnv returns a new Docker engine from the DOCKER_HOST and DOCKER_CERT_PATH
// environment variables.
func NewEnv() (runner.Engine, error) {
	config, err := dockerclient.TLSConfigFromCertPath(dockerCert)
	if err == nil && dockerTLS != "1" {
		config.InsecureSkipVerify = true
	}
	client, err := dockerclient.NewDockerClient(dockerHost, config)
	if err != nil {
		return nil, err
	}
	return New(client), nil
}

// MustEnv returns a new Docker engine from the DOCKER_HOST and DOCKER_CERT_PATH
// environment variables. Errors creating the Docker engine will panic.
func MustEnv() runner.Engine {
	engine, err := NewEnv()
	if err != nil {
		panic(err)
	}
	return engine
}
