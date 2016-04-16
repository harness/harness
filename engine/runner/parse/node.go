package parse

const (
	NodeList     = "list"
	NodeDefer    = "defer"
	NodeError    = "error"
	NodeRecover  = "recover"
	NodeParallel = "parallel"
	NodeRun      = "run"
)

// NodeType identifies the type of a parse tree node.
type NodeType string

// Type returns itself and provides an easy default implementation
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
	Validate() error
}
