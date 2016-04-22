package builtin

import (
	"path/filepath"
	"strings"

	"github.com/drone/drone/engine/compiler/parse"
)

type normalizeOp struct {
	visitor
	namespace string
}

// NewNormalizeOp returns a transformer that normalizes the container image
// names and plugin names to their fully qualified values.
func NewNormalizeOp(namespace string) Visitor {
	return &normalizeOp{
		namespace: namespace,
	}
}

func (v *normalizeOp) VisitContainer(node *parse.ContainerNode) error {
	v.normalizeName(node)
	v.normalizeImage(node)
	switch node.NodeType {
	case parse.NodePlugin, parse.NodeCache, parse.NodeClone:
		v.normalizePlugin(node)
	}
	return nil
}

// normalize the container image to the fully qualified name.
func (v *normalizeOp) normalizeImage(node *parse.ContainerNode) {
	if strings.Contains(node.Container.Image, ":") {
		return
	}
	node.Container.Image = node.Container.Image + ":latest"
}

// normalize the plugin entrypoint and command values.
func (v *normalizeOp) normalizePlugin(node *parse.ContainerNode) {
	if strings.Contains(node.Container.Image, "/") {
		return
	}
	if strings.Contains(node.Container.Image, "_") {
		node.Container.Image = strings.Replace(node.Container.Image, "_", "-", -1)
	}
	node.Container.Image = filepath.Join(v.namespace, node.Container.Image)
}

// normalize the container name to ensrue a value is set.
func (v *normalizeOp) normalizeName(node *parse.ContainerNode) {
	if node.Container.Name != "" {
		return
	}

	parts := strings.Split(node.Container.Image, "/")
	if len(parts) != 0 {
		node.Container.Name = parts[len(parts)-1]
	}
	parts = strings.Split(node.Container.Image, ":")
	if len(parts) != 0 {
		node.Container.Name = parts[0]
	}
}
