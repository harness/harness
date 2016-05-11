package build

import "github.com/drone/drone/yaml"

// Config defines the configuration for creating the Pipeline.
type Config struct {
	Engine Engine

	// Buffer defines the size of the buffer for the channel to which the
	// console output is streamed.
	Buffer uint
}

// Pipeline creates a build Pipeline using the specific configuration for
// the given Yaml specification.
func (c *Config) Pipeline(spec *yaml.Config) *Pipeline {

	pipeline := Pipeline{
		engine: c.Engine,
		pipe:   make(chan *Line, c.Buffer),
		next:   make(chan error),
		done:   make(chan error),
	}

	var containers []*yaml.Container
	containers = append(containers, spec.Services...)
	containers = append(containers, spec.Pipeline...)

	for _, c := range containers {
		if c.Disabled {
			continue
		}
		next := &element{Container: c}
		if pipeline.head == nil {
			pipeline.head = next
			pipeline.tail = next
		} else {
			pipeline.tail.next = next
			pipeline.tail = next
		}
	}

	go func() {
		pipeline.next <- nil
	}()

	return &pipeline
}
