package builtin

import (
	"path/filepath"

	"github.com/drone/drone/engine/compiler/parse"
)

type escalateOp struct {
	visitor
	plugins []string
}

// NewEscalateOp returns a transformer that configures plugins to automatically
// execute in privileged mode. This is intended for plugins running dind.
func NewEscalateOp(plugins []string) Visitor {
	return &escalateOp{
		plugins: plugins,
	}
}

func (v *escalateOp) VisitContainer(node *parse.ContainerNode) error {
	for _, pattern := range v.plugins {
		ok, _ := filepath.Match(pattern, node.Container.Image)
		if ok {
			node.Container.Privileged = true
		}
	}
	return nil
}
