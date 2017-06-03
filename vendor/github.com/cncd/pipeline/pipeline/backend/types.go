package backend

type (
	// Config defines the runtime configuration of a pipeline.
	Config struct {
		Stages   []*Stage   `json:"pipeline"` // pipeline stages
		Networks []*Network `json:"networks"` // network definitions
		Volumes  []*Volume  `json:"volumes"`  // volume definitions
		Secrets  []*Secret  `json:"secrets"`  // secret definitions
	}

	// Stage denotes a collection of one or more steps.
	Stage struct {
		Name  string  `json:"name,omitempty"`
		Alias string  `json:"alias,omitempty"`
		Steps []*Step `json:"steps,omitempty"`
	}

	// Step defines a container process.
	Step struct {
		Name         string            `json:"name"`
		Alias        string            `json:"alias,omitempty"`
		Image        string            `json:"image,omitempty"`
		Pull         bool              `json:"pull,omitempty"`
		Detached     bool              `json:"detach,omitempty"`
		Privileged   bool              `json:"privileged,omitempty"`
		WorkingDir   string            `json:"working_dir,omitempty"`
		Environment  map[string]string `json:"environment,omitempty"`
		Labels       map[string]string `json:"labels,omitempty"`
		Entrypoint   []string          `json:"entrypoint,omitempty"`
		Command      []string          `json:"command,omitempty"`
		ExtraHosts   []string          `json:"extra_hosts,omitempty"`
		Volumes      []string          `json:"volumes,omitempty"`
		Devices      []string          `json:"devices,omitempty"`
		Networks     []Conn            `json:"networks,omitempty"`
		DNS          []string          `json:"dns,omitempty"`
		DNSSearch    []string          `json:"dns_search,omitempty"`
		MemSwapLimit int64             `json:"memswap_limit,omitempty"`
		MemLimit     int64             `json:"mem_limit,omitempty"`
		ShmSize      int64             `json:"shm_size,omitempty"`
		CPUQuota     int64             `json:"cpu_quota,omitempty"`
		CPUShares    int64             `json:"cpu_shares,omitempty"`
		CPUSet       string            `json:"cpu_set,omitempty"`
		OnFailure    bool              `json:"on_failure,omitempty"`
		OnSuccess    bool              `json:"on_success,omitempty"`
		AuthConfig   Auth              `json:"auth_config,omitempty"`
		NetworkMode  string            `json:"network_mode,omitempty"`
	}

	// Auth defines registry authentication credentials.
	Auth struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Email    string `json:"email,omitempty"`
	}

	// Conn defines a container network connection.
	Conn struct {
		Name    string   `json:"name"`
		Aliases []string `json:"aliases"`
	}

	// Network defines a container network.
	Network struct {
		Name       string            `json:"name,omitempty"`
		Driver     string            `json:"driver,omitempty"`
		DriverOpts map[string]string `json:"driver_opts,omitempty"`
	}

	// Volume defines a container volume.
	Volume struct {
		Name       string            `json:"name,omitempty"`
		Driver     string            `json:"driver,omitempty"`
		DriverOpts map[string]string `json:"driver_opts,omitempty"`
	}

	// Secret defines a runtime secret
	Secret struct {
		Name  string `json:"name,omitempty"`
		Value string `json:"value,omitempty"`
		Mount string `json:"mount,omitempty"`
		Mask  bool   `json:"mask,omitempty"`
	}

	// State defines a container state.
	State struct {
		// Container exit code
		ExitCode int `json:"exit_code"`
		// Container exited, true or false
		Exited bool `json:"exited"`
		// Container is oom killed, true or false
		OOMKilled bool `json:"oom_killed"`
	}

	// // State defines the pipeline and process state.
	// State struct {
	// 	Pipeline struct {
	// 		// Current pipeline step
	// 		Step *Step `json:"step"`
	// 		// Current pipeline error state
	// 		Error error `json:"error"`
	// 	}
	//
	// 	Process struct {
	// 		// Container exit code
	// 		ExitCode int `json:"exit_code"`
	// 		// Container exited, true or false
	// 		Exited bool `json:"exited"`
	// 		// Container is oom killed, true or false
	// 		OOMKilled bool `json:"oom_killed"`
	// 	}
	// }
)
