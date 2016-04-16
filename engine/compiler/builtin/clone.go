package builtin

import (
	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"
)

type cloneOp struct {
	visitor
	plugin string
	enable bool
}

// NewCloneOp returns a transformer that configures the default clone plugin.
func NewCloneOp(plugin string, enable bool) Visitor {
	return &cloneOp{
		enable: enable,
		plugin: plugin,
	}
}

func (v *cloneOp) VisitContainer(node *parse.ContainerNode) error {
	if node.Type() != parse.NodeClone {
		return nil
	}
	if v.enable == false {
		node.Disabled = true
		return nil
	}

	if node.Container.Name == "" {
		node.Container.Name = "clone"
	}
	if node.Container.Image == "" {
		node.Container.Image = v.plugin
	}

	// discard any other cache properties except the image name.
	// everything else is discard for security reasons.
	node.Container = runner.Container{
		Name: node.Container.Name,
		Image: node.Container.Image,
	}
	return nil
}
