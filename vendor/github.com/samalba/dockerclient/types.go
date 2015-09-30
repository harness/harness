package dockerclient

import (
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/units"
)

type ContainerConfig struct {
	Hostname        string
	Domainname      string
	User            string
	AttachStdin     bool
	AttachStdout    bool
	AttachStderr    bool
	ExposedPorts    map[string]struct{}
	Tty             bool
	OpenStdin       bool
	StdinOnce       bool
	Env             []string
	Cmd             []string
	Image           string
	Volumes         map[string]struct{}
	VolumeDriver    string
	WorkingDir      string
	Entrypoint      []string
	NetworkDisabled bool
	MacAddress      string
	OnBuild         []string
	Labels          map[string]string

	// FIXME: The following fields have been removed since API v1.18
	Memory     int64
	MemorySwap int64
	CpuShares  int64
	Cpuset     string
	PortSpecs  []string

	// This is used only by the create command
	HostConfig HostConfig
}

type HostConfig struct {
	Binds           []string
	ContainerIDFile string
	LxcConf         []map[string]string
	Memory          int64
	MemorySwap      int64
	CpuShares       int64
	CpuPeriod       int64
	CpusetCpus      string
	CpusetMems      string
	CpuQuota        int64
	BlkioWeight     int64
	OomKillDisable  bool
	Privileged      bool
	PortBindings    map[string][]PortBinding
	Links           []string
	PublishAllPorts bool
	Dns             []string
	DnsSearch       []string
	ExtraHosts      []string
	VolumesFrom     []string
	Devices         []DeviceMapping
	NetworkMode     string
	IpcMode         string
	PidMode         string
	UTSMode         string
	CapAdd          []string
	CapDrop         []string
	RestartPolicy   RestartPolicy
	SecurityOpt     []string
	ReadonlyRootfs  bool
	Ulimits         []Ulimit
	LogConfig       LogConfig
	CgroupParent    string
}

type DeviceMapping struct {
	PathOnHost        string `json:"PathOnHost"`
	PathInContainer   string `json:"PathInContainer"`
	CgroupPermissions string `json:"CgroupPermissions"`
}

type ExecConfig struct {
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	Cmd          []string
	Container    string
	Detach       bool
}

type LogOptions struct {
	Follow     bool
	Stdout     bool
	Stderr     bool
	Timestamps bool
	Tail       int64
}

type MonitorEventsFilters struct {
	Event     string `json:",omitempty"`
	Image     string `json:",omitempty"`
	Container string `json:",omitempty"`
}

type MonitorEventsOptions struct {
	Since   int
	Until   int
	Filters *MonitorEventsFilters `json:",omitempty"`
}

type RestartPolicy struct {
	Name              string
	MaximumRetryCount int64
}

type PortBinding struct {
	HostIp   string
	HostPort string
}

type State struct {
	Running    bool
	Paused     bool
	Restarting bool
	OOMKilled  bool
	Dead       bool
	Pid        int
	ExitCode   int
	Error      string // contains last known error when starting the container
	StartedAt  time.Time
	FinishedAt time.Time
	Ghost      bool
}

// String returns a human-readable description of the state
// Stoken from docker/docker/daemon/state.go
func (s *State) String() string {
	if s.Running {
		if s.Paused {
			return fmt.Sprintf("Up %s (Paused)", units.HumanDuration(time.Now().UTC().Sub(s.StartedAt)))
		}
		if s.Restarting {
			return fmt.Sprintf("Restarting (%d) %s ago", s.ExitCode, units.HumanDuration(time.Now().UTC().Sub(s.FinishedAt)))
		}

		return fmt.Sprintf("Up %s", units.HumanDuration(time.Now().UTC().Sub(s.StartedAt)))
	}

	if s.Dead {
		return "Dead"
	}

	if s.FinishedAt.IsZero() {
		return ""
	}

	return fmt.Sprintf("Exited (%d) %s ago", s.ExitCode, units.HumanDuration(time.Now().UTC().Sub(s.FinishedAt)))
}

// StateString returns a single string to describe state
// Stoken from docker/docker/daemon/state.go
func (s *State) StateString() string {
	if s.Running {
		if s.Paused {
			return "paused"
		}
		if s.Restarting {
			return "restarting"
		}
		return "running"
	}

	if s.Dead {
		return "dead"
	}

	return "exited"
}

type ImageInfo struct {
	Architecture    string
	Author          string
	Comment         string
	Config          *ContainerConfig
	Container       string
	ContainerConfig *ContainerConfig
	Created         time.Time
	DockerVersion   string
	Id              string
	Os              string
	Parent          string
	Size            int64
	VirtualSize     int64
}

type ContainerInfo struct {
	Id              string
	Created         string
	Path            string
	Name            string
	Args            []string
	ExecIDs         []string
	Config          *ContainerConfig
	State           *State
	Image           string
	NetworkSettings struct {
		IPAddress   string `json:"IpAddress"`
		IPPrefixLen int    `json:"IpPrefixLen"`
		Gateway     string
		Bridge      string
		Ports       map[string][]PortBinding
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	HostConfig     *HostConfig
}

type ContainerChanges struct {
	Path string
	Kind int
}

type Port struct {
	IP          string
	PrivatePort int
	PublicPort  int
	Type        string
}

type Container struct {
	Id         string
	Names      []string
	Image      string
	Command    string
	Created    int64
	Status     string
	Ports      []Port
	SizeRw     int64
	SizeRootFs int64
	Labels     map[string]string
}

type Event struct {
	Id     string
	Status string
	From   string
	Time   int64
}

type Version struct {
	ApiVersion    string
	Arch          string
	GitCommit     string
	GoVersion     string
	KernelVersion string
	Os            string
	Version       string
}

type RespContainersCreate struct {
	Id       string
	Warnings []string
}

type Image struct {
	Created     int64
	Id          string
	ParentId    string
	RepoTags    []string
	Size        int64
	VirtualSize int64
}

// Info is the struct returned by /info
// The API is currently in flux, so Debug, MemoryLimit, SwapLimit, and
// IPv4Forwarding are interfaces because in docker 1.6.1 they are 0 or 1 but in
// master they are bools.
type Info struct {
	ID                 string
	Containers         int64
	Driver             string
	DriverStatus       [][]string
	ExecutionDriver    string
	Images             int64
	KernelVersion      string
	OperatingSystem    string
	NCPU               int64
	MemTotal           int64
	Name               string
	Labels             []string
	Debug              interface{}
	NFd                int64
	NGoroutines        int64
	SystemTime         string
	NEventsListener    int64
	InitPath           string
	InitSha1           string
	IndexServerAddress string
	MemoryLimit        interface{}
	SwapLimit          interface{}
	IPv4Forwarding     interface{}
	BridgeNfIptables   bool
	BridgeNfIp6tables  bool
	DockerRootDir      string
	HttpProxy          string
	HttpsProxy         string
	NoProxy            string
}

type ImageDelete struct {
	Deleted  string
	Untagged string
}

type EventOrError struct {
	Event
	Error error
}

type WaitResult struct {
	ExitCode int
	Error    error
}

type decodingResult struct {
	result interface{}
	err    error
}

// The following are types for the API stats endpoint
type ThrottlingData struct {
	// Number of periods with throttling active
	Periods uint64 `json:"periods"`
	// Number of periods when the container hit its throttling limit.
	ThrottledPeriods uint64 `json:"throttled_periods"`
	// Aggregate time the container was throttled for in nanoseconds.
	ThrottledTime uint64 `json:"throttled_time"`
}

type CpuUsage struct {
	// Total CPU time consumed.
	// Units: nanoseconds.
	TotalUsage uint64 `json:"total_usage"`
	// Total CPU time consumed per core.
	// Units: nanoseconds.
	PercpuUsage []uint64 `json:"percpu_usage"`
	// Time spent by tasks of the cgroup in kernel mode.
	// Units: nanoseconds.
	UsageInKernelmode uint64 `json:"usage_in_kernelmode"`
	// Time spent by tasks of the cgroup in user mode.
	// Units: nanoseconds.
	UsageInUsermode uint64 `json:"usage_in_usermode"`
}

type CpuStats struct {
	CpuUsage       CpuUsage       `json:"cpu_usage"`
	SystemUsage    uint64         `json:"system_cpu_usage"`
	ThrottlingData ThrottlingData `json:"throttling_data,omitempty"`
}

type NetworkStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

type MemoryStats struct {
	Usage    uint64            `json:"usage"`
	MaxUsage uint64            `json:"max_usage"`
	Stats    map[string]uint64 `json:"stats"`
	Failcnt  uint64            `json:"failcnt"`
	Limit    uint64            `json:"limit"`
}

type BlkioStatEntry struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

type BlkioStats struct {
	// number of bytes tranferred to and from the block device
	IoServiceBytesRecursive []BlkioStatEntry `json:"io_service_bytes_recursive"`
	IoServicedRecursive     []BlkioStatEntry `json:"io_serviced_recursive"`
	IoQueuedRecursive       []BlkioStatEntry `json:"io_queue_recursive"`
	IoServiceTimeRecursive  []BlkioStatEntry `json:"io_service_time_recursive"`
	IoWaitTimeRecursive     []BlkioStatEntry `json:"io_wait_time_recursive"`
	IoMergedRecursive       []BlkioStatEntry `json:"io_merged_recursive"`
	IoTimeRecursive         []BlkioStatEntry `json:"io_time_recursive"`
	SectorsRecursive        []BlkioStatEntry `json:"sectors_recursive"`
}

type Stats struct {
	Read         time.Time    `json:"read"`
	NetworkStats NetworkStats `json:"network,omitempty"`
	CpuStats     CpuStats     `json:"cpu_stats,omitempty"`
	MemoryStats  MemoryStats  `json:"memory_stats,omitempty"`
	BlkioStats   BlkioStats   `json:"blkio_stats,omitempty"`
}

type Ulimit struct {
	Name string `json:"name"`
	Soft uint64 `json:"soft"`
	Hard uint64 `json:"hard"`
}

type LogConfig struct {
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

type BuildImage struct {
	Config         *ConfigFile
	DockerfileName string
	Context        io.Reader
	RemoteURL      string
	RepoName       string
	SuppressOutput bool
	NoCache        bool
	Remove         bool
	ForceRemove    bool
	Pull           bool
	Memory         int64
	MemorySwap     int64
	CpuShares      int64
	CpuPeriod      int64
	CpuQuota       int64
	CpuSetCpus     string
	CpuSetMems     string
	CgroupParent   string
}
