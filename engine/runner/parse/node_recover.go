package parse

import "fmt"

type RecoverNode struct {
	NodeType `json:"type"`

	Body Node `json:"body"` // evaluate node and catch all errors.
}

func NewRecoverNode() *RecoverNode {
	return &RecoverNode{NodeType: NodeRecover}
}

func (n *RecoverNode) SetBody(node Node) *RecoverNode {
	n.Body = node
	return n
}

func (n *RecoverNode) Validate() error {
	switch {
	case n.NodeType != NodeRecover:
		return fmt.Errorf("Recover Node uses an invalid type")
	case n.Body == nil:
		return fmt.Errorf("Recover Node body is empty")
	default:
		return nil
	}
}
