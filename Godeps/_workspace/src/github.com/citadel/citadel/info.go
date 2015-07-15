package citadel

type (
	ClusterInfo struct {
		Cpus           float64 `json:"cpus,omitempty"`
		Memory         float64 `json:"memory,omitempty"`
		ContainerCount int     `json:"container_count,omitempty"`
		EngineCount    int     `json:"engine_count,omitempty"`
		ImageCount     int     `json:"image_count,omitempty"`
		ReservedCpus   float64 `json:"reserved_cpus,omitempty"`
		ReservedMemory float64 `json:"reserved_memory,omitempty"`
	}
)
