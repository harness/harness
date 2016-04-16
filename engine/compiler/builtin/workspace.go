package builtin

import (
	"path/filepath"

	"github.com/drone/drone/engine/compiler/parse"
)

type workspaceOp struct {
	visitor
	base string
	path string
}

// NewWorkspaceOp returns a transformer that provides a default workspace paths,
// including the base path (mounted as a volume) and absolute path where the
// code is cloned.
func NewWorkspaceOp(base, path string) Visitor {
	return &workspaceOp{
		base: base,
		path: path,
	}
}

func (v *workspaceOp) VisitRoot(node *parse.RootNode) error {
	if node.Base == "" {
		node.Base = v.base
	}
	if node.Path == "" {
		node.Path = v.path
	}
	if !filepath.IsAbs(node.Path) {
		node.Path = filepath.Join(
			node.Base,
			node.Path,
		)
	}
	return nil
}

func (v *workspaceOp) VisitContainer(node *parse.ContainerNode) error {
	if node.NodeType == parse.NodeService {
		// we must not override the default working
		// directory of service containers. All other
		// container should launch in the workspace
		return nil
	}
	node.Container.WorkingDir = node.Root().Path
	return nil
}
