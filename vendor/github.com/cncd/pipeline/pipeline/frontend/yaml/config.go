package yaml

import (
	"io"
	"io/ioutil"
	"os"

	libcompose "github.com/docker/libcompose/yaml"
	"gopkg.in/yaml.v2"
)

type (
	// Config defines a pipeline configuration.
	Config struct {
		Cache     libcompose.Stringorslice
		Platform  string
		Branches  Constraint
		Workspace Workspace
		Clone     Containers
		Pipeline  Containers
		Services  Containers
		Networks  Networks
		Volumes   Volumes
		Labels    libcompose.SliceorMap
	}

	// Workspace defines a pipeline workspace.
	Workspace struct {
		Base string
		Path string
	}
)

// Parse parses the configuration from bytes b.
func Parse(r io.Reader) (*Config, error) {
	out, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseBytes(out)
}

// ParseBytes parses the configuration from bytes b.
func ParseBytes(b []byte) (*Config, error) {
	out := new(Config)
	err := yaml.Unmarshal(b, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ParseString parses the configuration from string s.
func ParseString(s string) (*Config, error) {
	return ParseBytes(
		[]byte(s),
	)
}

// ParseFile parses the configuration from path p.
func ParseFile(p string) (*Config, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}
