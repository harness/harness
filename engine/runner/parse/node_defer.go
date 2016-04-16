package parse

import "fmt"

// DeferNode executes the child node, and then executes the deffered node.
// The deffered node is guaranteed to execute, even when the child node fails.
type DeferNode struct {
	NodeType `json:"type"`

	Body  Node `json:"body"`  // evaluate node
	Defer Node `json:"defer"` // defer evaluation of node.
}

// NewDeferNode returns a new DeferNode.
func NewDeferNode() *DeferNode {
	return &DeferNode{NodeType: NodeDefer}
}

func (n *DeferNode) SetBody(node Node) *DeferNode {
	n.Body = node
	return n
}

func (n *DeferNode) SetDefer(node Node) *DeferNode {
	n.Defer = node
	return n
}

func (n *DeferNode) Validate() error {
	switch {
	case n.NodeType != NodeDefer:
		return fmt.Errorf("Defer Node uses an invalid type")
	case n.Body == nil:
		return fmt.Errorf("Defer Node body is empty")
	case n.Defer == nil:
		return fmt.Errorf("Defer Node defer is empty")
	default:
		return nil
	}
}
