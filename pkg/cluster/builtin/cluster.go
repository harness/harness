package builtin

import (
	"os"
	"regexp"
	"strconv"
	"io"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/cluster"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/scheduler"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
)

const (
	// Default docker host address
	DefaultHost = "unix:///var/run/docker.sock"
	// Multiple Dockers ENV variable prefix
	DockerPrefix = "DOCKER_HOST_"
	// Default docker host limits
	DefaultMemory = 2048
	DefaultCPUs = 1
	// Default job container limits
	// DefaultContainerCPUs = os.Getenv("DOCKER_CONTAINER_CPU")
	// DefaultContainerMemory = os.Getenv("DOCKER_CONTAINER_MEM")
)

type Manager struct {
	cluster *cluster.Cluster
	tlc     *tls.Config
}

func GetTLSConfig() *tls.Config {
	var tlc *tls.Config
	// Docker TLS variables
	DockerHostCa := os.Getenv("DOCKER_CA")
	DockerHostKey := os.Getenv("DOCKER_KEY")
	DockerHostCert := os.Getenv("DOCKER_CERT")
	// create the Docket client TLS config
	if len(DockerHostCert) > 0 && len(DockerHostKey) > 0 && len(DockerHostCa) > 0 {
		cert, err := tls.LoadX509KeyPair(DockerHostCert, DockerHostKey)
		if err != nil {
			log.Errorf("failure to load SSL cert and key. %s", err)
		}
		caCert, err := ioutil.ReadFile(DockerHostCa)
		if err != nil {
			log.Errorf("failure to load SSL CA cert. %s", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlc = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
		}
	}
	return tlc
}

func GetFloatFromEnv(key string) (value float64, found bool) {
	val := os.Getenv(key)
	if len(val) > 0 {
		float, error := strconv.ParseFloat(val, 10)
		if error == nil {
			return float, true
		}
	}
	return 0, false
}

func New() *Manager {
	log.Errorln("Start new cluster scheduler")
	c, err := cluster.New(scheduler.NewResourceManager())
	if err != nil {
		panic(err)
	}
	log.Errorln("Register label scheduler")
	if err := c.RegisterScheduler("drone_internal", &scheduler.LabelScheduler{}); err != nil {
		panic(err)
	}
	manager := &Manager{
		cluster: c,
		tlc: GetTLSConfig(),
	}
	manager.CollectDockers()
	return manager
}

func (c *Manager) CollectDockers() error {
	r, _ := regexp.Compile(`\A` + DockerPrefix + `(\d+)\z`)
	for _, key := range os.Environ() {
		matches := r.FindStringSubmatch(key)
		if len(matches) == 2 {
			c.AddDocker(matches[1])
		}
	}
	if len(c.cluster.Engines()) == 0 {
		log.Errorln("Set default docker host")
		c.AddDefaultDocker()
	}
	return nil
}

func (c *Manager) AddDocker(index string) error {
	prefix := DockerPrefix + index
	addr := os.Getenv(prefix)
	label := os.Getenv(prefix + "_LABEL")
	if len(label) == 0 {
		label = prefix
	}
	engine := &citadel.Engine{
		ID: label,
		Addr: addr,
		Labels: []string{label},
	}
	cpu_num, found := GetFloatFromEnv(prefix + "_CPU")
	if found {
		engine.Cpus = cpu_num
	}
	mem_num, found := GetFloatFromEnv(prefix + "_MEM")
	if found {
		engine.Memory = mem_num
	}
	return c.AddEngine(engine)
}

func (c *Manager) AddDefaultDocker() error {
	addr := os.Getenv("DOCKER_HOST")
	if len(addr) == 0 {
		addr = DefaultHost
	}
	engine := &citadel.Engine{
		ID: "Default",
		Addr: addr,
		Labels: []string{"Default"},
		Cpus: 1,
		Memory: 2048,
	}
	return c.AddEngine(engine)
}

func (c *Manager) AddEngine(engine *citadel.Engine) error {
	if err := engine.Connect(c.tlc); err != nil {
		log.Errorf("Could not connect to docker: %s", err)
		return err
	}
	c.cluster.AddEngine(engine)
	return nil
}

func (c *Manager) ClusterStats() *citadel.ClusterInfo {
	return c.cluster.ClusterInfo()
}

func (c *Manager) Start(image *citadel.Image, pull bool) (*citadel.Container, error) {
	return c.cluster.Start(image, pull)
}

func (c *Manager) StopAndKillContainer(container *citadel.Container) error {
	err := c.cluster.Stop(container)
	if err != nil {
		return err
	}
	err = c.cluster.Kill(container, 9)
	if err != nil {
		return err
	}
	return nil
}

func (c *Manager) RemoveContainer(container *citadel.Container) error {
	err := c.cluster.Remove(container)
	if err != nil {
		return err
	}
	return nil
}

func (c *Manager) Logs(container *citadel.Container, follow bool) (log io.ReadCloser, err error) {
	log, err = c.cluster.Logs(container, true, true, follow)
	return
}

func (c *Manager) ContainerInfo(container *citadel.Container) (*dockerclient.ContainerInfo, error) {
	return container.Engine.Info(container)
}

func (c *Manager) FindContainerByName(name string) *citadel.Container {
	containers := c.cluster.ListContainers(false, false, "")
	for _, container := range containers {
		if container.Name == name {
			return container
		}
	}
	return nil
}