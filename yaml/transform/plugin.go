package transform

import (
	"path/filepath"

	"github.com/drone/drone/yaml"
)

// PluginDisable is a transform function that alters the Yaml configuration to
// disables plugins. This is intended for use when executing the pipeline
// locally on your own computer.
func PluginDisable(conf *yaml.Config, patterns []string) error {
	for _, container := range conf.Pipeline {
		if len(container.Commands) != 0 { // skip build steps
			continue
		}
		var match bool
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(pattern, container.Name); ok {
				match = true
				break
			}
		}
		if !match {
			container.Disabled = true
		}
	}
	return nil
}

// PluginParams is a transform function that alters the Yaml configuration to
// include plugin vargs parameters as environment variables.
func PluginParams(conf *yaml.Config) error {
	for _, container := range conf.Pipeline {
		if len(container.Vargs) == 0 {
			continue
		}
		if container.Environment == nil {
			container.Environment = map[string]string{}
		}
		err := argsToEnv(container.Vargs, container.Environment)
		if err != nil {
			return err
		}
	}
	return nil
}
