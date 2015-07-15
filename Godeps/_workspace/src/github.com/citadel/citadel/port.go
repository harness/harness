package citadel

type Port struct {
	Proto         string `json:"proto,omitempty"`
	HostIp        string `json:"host_ip,omitempty"`
	Port          int    `json:"port,omitempty"`
	ContainerPort int    `json:"container_port,omitempty"`
}
