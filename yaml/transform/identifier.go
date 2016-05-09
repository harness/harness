package transform

import (
	"encoding/base64"
	"fmt"

	"github.com/drone/drone/yaml"

	"github.com/gorilla/securecookie"
)

// Identifier transforms the container steps in the Yaml and assigns a unique
// container identifier.
func Identifier(c *yaml.Config) error {

	// creates a random prefix for the build
	rand := base64.RawURLEncoding.EncodeToString(
		securecookie.GenerateRandomKey(8),
	)

	for i, step := range c.Services {
		step.ID = fmt.Sprintf("drone_%s_%d", rand, i)
	}

	for i, step := range c.Pipeline {
		step.ID = fmt.Sprintf("drone_%s_%d", rand, i+len(c.Services))
	}

	return nil
}
