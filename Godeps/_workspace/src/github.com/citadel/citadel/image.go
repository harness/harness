package citadel

import "fmt"

// Image is a template for running a docker container
type Image struct {
	// Name is the docker image to base the container off of
	Name string `json:"name,omitempty"`

	// Cpus is the number of cpu resources to give to the container
	Cpus float64 `json:"cpus,omitempty"`

	// Cpuset is the single or multiple set of cpus on which the container can run
	Cpuset string `json:"cpuset,omitempty"`

	// Memory is the amount of memory in MB for the container
	Memory float64 `json:"memory,omitempty"`

	// Entrypoint is the entrypoint in the container
	Entrypoint []string `json:"entrypoint,omitempty"`

	// Envionrment is the environment vars to set on the container
	Environment map[string]string `json:"environment,omitempty"`

	// Hostname is the host name to set for the container
	Hostname string `json:"hostname,omitempty"`

	// Domainname is the domain name to set for the container
	Domainname string `json:"domain,omitempty"`

	// Args are cli arguments to pass to the image
	Args []string `json:"args,omitempty"`

	// Type is the container type, often service, batch, etc...
	Type string `json:"type,omitempty"`

	// Labels are matched with constraints on the engines
	Labels []string `json:"labels,omitempty"`

	// BindPorts ensures that the container has exclusive access to the specified ports
	BindPorts []*Port `json:"bind_ports,omitempty"`

	// UserData is user defined data that is passed to the container
	UserData map[string][]string `json:"user_data,omitempty"`

	// Volumes are volumes on the same engine
	Volumes []string `json:"volumes,omitempty"`

	// Links are mappings to other containers running on the same engine
	Links map[string]string `json:"links,omitempty"`

	// RestartPolicy is the container restart policy if it exits
	RestartPolicy RestartPolicy `json:"restart_policy,omitempty"`

	// Publish tells the engine to expose ports the the container externally
	Publish bool `json:"publish,omitempty"`

	// Give extended privileges to this container, e.g. Docker-in-Docker
	Privileged bool `json:"privileged,omitempty"`

	// NetworkMode is the network mode for the container
	NetworkMode string `json:"network_mode,omitempty"`

	// ContainerName is the name set to the container
	ContainerName string `json:"container_name,omitempty"`
}

type RestartPolicy struct {
	Name              string `json:"name,omitempty"`
	MaximumRetryCount int64  `json:"maximum_retry,omitempty"`
}

func (i *Image) String() string {
	return fmt.Sprintf("image %s type %s cpus %f cpuset %s memory %f", i.Name, i.Type, i.Cpus, i.Cpuset, i.Memory)
}
