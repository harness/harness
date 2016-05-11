package transform

import (
	"path/filepath"

	"github.com/drone/drone/yaml"
)

// WorkspaceTransform transforms ...
func WorkspaceTransform(c *yaml.Config, base, path string) error {
	if c.Workspace == nil {
		c.Workspace = &yaml.Workspace{}
	}

	if c.Workspace.Base == "" {
		c.Workspace.Base = base
	}
	if c.Workspace.Path == "" {
		c.Workspace.Path = path
	}
	if !filepath.IsAbs(c.Workspace.Path) {
		c.Workspace.Path = filepath.Join(
			c.Workspace.Base,
			c.Workspace.Path,
		)
	}

	for _, p := range c.Pipeline {
		p.WorkingDir = c.Workspace.Path
	}
	return nil
}
