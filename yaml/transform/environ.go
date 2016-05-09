package transform

import "github.com/drone/drone/yaml"

// Environ transforms the steps in the Yaml pipeline to include runtime
// environment variables.
func Environ(c *yaml.Config, envs map[string]string) error {
	for _, p := range c.Pipeline {
		if p.Environment == nil {
			p.Environment = map[string]string{}
		}
		for k, v := range envs {
			if v == "" {
				continue
			}
			p.Environment[k] = v
		}
	}
	return nil
}
