package parse

import "fmt"

// ErrorNode executes the body node, and then executes the error node if
// the body node errors. This is similar to defer but only executes on error.
type ErrorNode struct {
	NodeType `json:"type"`

	Body  Node `json:"body"`  // evaluate node
	Defer Node `json:"defer"` // defer evaluation of node on error.
}

// NewErrorNode returns a new ErrorNode.
func NewErrorNode() *ErrorNode {
	return &ErrorNode{NodeType: NodeError}
}

func (n *ErrorNode) SetBody(node Node) *ErrorNode {
	n.Body = node
	return n
}

func (n *ErrorNode) SetDefer(node Node) *ErrorNode {
	n.Defer = node
	return n
}

func (n *ErrorNode) Validate() error {
	switch {
	case n.NodeType != NodeError:
		return fmt.Errorf("Error Node uses an invalid type")
	case n.Body == nil:
		return fmt.Errorf("Error Node body is empty")
	case n.Defer == nil:
		return fmt.Errorf("Error Node defer is empty")
	default:
		return nil
	}
}
