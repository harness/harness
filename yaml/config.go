package yaml

import "gopkg.in/yaml.v2"

// Workspace represents the build workspace.
type Workspace struct {
	Base string
	Path string
}

// Config represents the build configuration Yaml document.
type Config struct {
	Image     string
	Build     *Build
	Workspace *Workspace
	Pipeline  []*Container
	Services  []*Container
	Volumes   []*Volume
	Networks  []*Network
}

// ParseString parses the Yaml configuration document.
func ParseString(data string) (*Config, error) {
	return Parse([]byte(data))
}

// Parse parses Yaml configuration document.
func Parse(data []byte) (*Config, error) {
	v := struct {
		Image     string
		Build     *Build
		Workspace *Workspace
		Services  containerList
		Pipeline  containerList
		Networks  networkList
		Volumes   volumeList
	}{}

	err := yaml.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}

	for _, c := range v.Services.containers {
		c.Detached = true
	}

	return &Config{
		Image:     v.Image,
		Build:     v.Build,
		Workspace: v.Workspace,
		Services:  v.Services.containers,
		Pipeline:  v.Pipeline.containers,
		Networks:  v.Networks.networks,
		Volumes:   v.Volumes.volumes,
	}, nil
}

type config struct {
	Image     string
	Build     *Build
	Workspace *Workspace
	Services  containerList
	Pipeline  containerList
	Networks  networkList
	Volumes   volumeList
}
