package docker

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// These are structures copied from the Docker project.
// We avoid importing the libraries due to a CGO
// depenency on libdevmapper that we'd like to avoid.

type KeyValuePair struct {
	Key   string
	Value string
}

type HostConfig struct {
	Binds           []string
	ContainerIDFile string
	LxcConf         []KeyValuePair
	Privileged      bool
	PortBindings    map[Port][]PortBinding
	Links           []string
	PublishAllPorts bool
}

type Top struct {
	Titles    []string
	Processes [][]string
}

type Containers struct {
	ID         string `json:"Id"`
	Image      string
	Command    string
	Created    int64
	Status     string
	Ports      []Port
	SizeRw     int64
	SizeRootFs int64
	Names      []string
}

type Run struct {
	ID       string   `json:"Id"`
	Warnings []string `json:",omitempty"`
}

type Wait struct {
	StatusCode int
}

type State struct {
	Running    bool
	Pid        int
	ExitCode   int
	StartedAt  time.Time
	FinishedAt time.Time
	Ghost      bool
}

type PortBinding struct {
	HostIp   string
	HostPort string
}

// 80/tcp
type Port string

func (p Port) Proto() string {
	parts := strings.Split(string(p), "/")
	if len(parts) == 1 {
		return "tcp"
	}
	return parts[1]
}

func (p Port) Port() string {
	return strings.Split(string(p), "/")[0]
}

func (p Port) Int() int {
	i, err := parsePort(p.Port())
	if err != nil {
		panic(err)
	}
	return i
}

func parsePort(rawPort string) (int, error) {
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil {
		return 0, err
	}
	return int(port), nil
}

func NewPort(proto, port string) Port {
	return Port(fmt.Sprintf("%s/%s", port, proto))
}

type PortMapping map[string]string // Deprecated

type NetworkSettings struct {
	IPAddress   string
	IPPrefixLen int
	Gateway     string
	Bridge      string
	PortMapping map[string]PortMapping // Deprecated
	Ports       map[Port][]PortBinding
}

type Config struct {
	Hostname        string
	Domainname      string
	User            string
	Memory          int64 // Memory limit (in bytes)
	MemorySwap      int64 // Total memory usage (memory + swap); set `-1' to disable swap
	CpuShares       int64 // CPU shares (relative weight vs. other containers)
	AttachStdin     bool
	AttachStdout    bool
	AttachStderr    bool
	PortSpecs       []string // Deprecated - Can be in the format of 8080/tcp
	ExposedPorts    map[Port]struct{}
	Tty             bool // Attach standard streams to a tty, including stdin if it is not closed.
	OpenStdin       bool // Open stdin
	StdinOnce       bool // If true, close stdin after the 1 attached client disconnects.
	Env             []string
	Cmd             []string
	Dns             []string
	Image           string // Name of the image as it was passed by the operator (eg. could be symbolic)
	Volumes         map[string]struct{}
	VolumesFrom     string
	WorkingDir      string
	Entrypoint      []string
	NetworkDisabled bool
}

type Container struct {
	ID string

	Created time.Time

	Path string
	Args []string

	Config *Config
	State  State
	Image  string

	NetworkSettings *NetworkSettings

	SysInitPath    string
	ResolvConfPath string
	HostnamePath   string
	HostsPath      string
	Name           string
	Driver         string

	Volumes map[string]string
	// Store rw/ro in a separate structure to preserve reverse-compatibility on-disk.
	// Easier than migrating older container configs :)
	VolumesRW map[string]bool
}
