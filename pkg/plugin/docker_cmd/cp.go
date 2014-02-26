package docker_cmd

type Copy struct {
	Source      string `yaml:"source,omitempty"`
	Destination string `yaml:"destination,omitempty"`
}
