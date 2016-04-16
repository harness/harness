package builtin

import (
	"github.com/drone/drone/engine/compiler/parse"
)

type pullOp struct {
	visitor
	pull bool
}

// NewPullOp returns a transformer that configures plugins to automatically
// pull the latest images at runtime.
func NewPullOp(pull bool) Visitor {
	return &pullOp{
		pull: pull,
	}
}

func (v *pullOp) VisitContainer(node *parse.ContainerNode) error {
	switch node.NodeType {
	case parse.NodePlugin, parse.NodeCache, parse.NodeClone:
		node.Container.Pull = v.pull
	}
	return nil
}
