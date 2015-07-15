package citadel

import "fmt"

// Container is a running instance
type Container struct {
	// ID is the container's id
	ID string `json:"id,omitempty"`

	// Name is the container's name
	Name string `json:"name,omitempty"`

	// Image is the configuration from which the container was created
	Image *Image `json:"image,omitempty"`

	// Engine is the engine that is runnnig the container
	Engine *Engine `json:"engine,omitempty"`

	// State is the container state ( running stopped )
	State string `json:"state,omitempty"`

	// Ports are the public port mappings for the container
	Ports []*Port `json:"ports,omitempty"`
}

func (c *Container) String() string {
	name := c.ID
	if c.Name != "" {
		name = c.Name
	}

	return fmt.Sprintf("container %s Image %s Engine %s", name, c.Image.Name, c.Engine.ID)
}
