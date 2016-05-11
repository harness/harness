package docker

import (
	"github.com/drone/drone/build"
	"github.com/samalba/dockerclient"
)

// NewClient returns a new Docker engine using the provided Docker client.
func NewClient(client dockerclient.Client) build.Engine {
	return &dockerEngine{client}
}

// New returns a new Docker engine from the provided DOCKER_HOST and
// DOCKER_CERT_PATH environment variables.
func New(host, cert string, tls bool) (build.Engine, error) {
	config, err := dockerclient.TLSConfigFromCertPath(cert)
	if err == nil && tls {
		config.InsecureSkipVerify = true
	}
	client, err := dockerclient.NewDockerClient(host, config)
	if err != nil {
		return nil, err
	}
	return NewClient(client), nil
}
