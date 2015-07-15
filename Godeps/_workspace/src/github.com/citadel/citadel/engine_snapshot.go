package citadel

type EngineSnapshot struct {
	// ID is the engines id
	ID string `json:"id,omitempty"`

	Cpus float64 `json:"cpus,omitempty"`

	Memory float64 `json:"memory,omitempty"`

	// ReservedCpus is the total amount of cpus that is reserved
	ReservedCpus float64 `json:"reserved_cpus,omitempty"`

	// ReservedMemory is the total amount of memory that is reserved
	ReservedMemory float64 `json:"reserved_memory,omitempty"`

	// CurrentMemory is the current system's used memory at the time of the snapshot
	CurrentMemory float64 `json:"current_memory,omitempty"`

	// CurrentCpu is the current system's cpu usage at the time of the snapshot
	CurrentCpu float64 `json:"current_cpu,omitempty"`
}
