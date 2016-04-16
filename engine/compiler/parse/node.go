package parse

const (
	NodeBuild     = "build"
	NodeCache     = "cache"
	NodeClone     = "clone"
	NodeContainer = "container"
	NodeNetwork   = "network"
	NodePlugin    = "plugin"
	NodeRoot      = "root"
	NodeService   = "service"
	NodeShell     = "shell"
	NodeVolume    = "volume"
)

// NodeType identifies the type of parse tree node.
type NodeType string

// Type returns itself an provides an easy default implementation.
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

// String returns the string value of the Node type.
func (t NodeType) String() string {
	return string(t)
}

// A Node is an element in the parse tree.
type Node interface {
	Type() NodeType
	Root() *RootNode
}
