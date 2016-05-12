package transform

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/yaml"
)

// ChangeFilter is a transform function that alters status constraints set to
// change and replaces with the opposite of the prior status.
func ChangeFilter(conf *yaml.Config, prev string) {
	for _, step := range conf.Pipeline {
		for i, status := range step.Constraints.Status.Include {
			if status != "change" && status != "changed" {
				continue
			}
			if prev == model.StatusFailure {
				step.Constraints.Status.Include[i] = model.StatusSuccess
			} else {
				step.Constraints.Status.Include[i] = model.StatusFailure
			}
		}
	}
}

// DefaultFilter is a transform function that applies default Filters to each
// step in the Yaml specification file.
func DefaultFilter(conf *yaml.Config) {
	for _, step := range conf.Pipeline {
		defaultStatus(step)
		defaultEvent(step)
	}
}

// defaultStatus sets default status conditions.
func defaultStatus(c *yaml.Container) {
	if !isEmpty(c.Constraints.Status) {
		return
	}
	c.Constraints.Status.Include = []string{
		model.StatusSuccess,
	}
}

// defaultEvent sets default event conditions.
func defaultEvent(c *yaml.Container) {
	if !isEmpty(c.Constraints.Event) {
		return
	}

	if isPlugin(c) && !isClone(c) {
		c.Constraints.Event.Exclude = []string{
			model.EventPull,
		}
	}
}

// helper function returns true if the step is a clone step.
func isEmpty(c yaml.Constraint) bool {
	return len(c.Include) == 0 && len(c.Exclude) == 0
}

// helper function returns true if the step is a plugin step.
func isPlugin(c *yaml.Container) bool {
	return len(c.Commands) == 0 || len(c.Vargs) != 0
}

// helper function returns true if the step is a command step.
func isCommand(c *yaml.Container) bool {
	return len(c.Commands) != 0
}

// helper function returns true if the step is a clone step.
func isClone(c *yaml.Container) bool {
	return c.Name == "clone"
}
