package parse

import "fmt"

// ListNode serially executes a list of child nodes.
type ListNode struct {
	NodeType `json:"type"`

	// Body is the list of child nodes
	Body []Node `json:"body"`
}

// NewListNode returns a new ListNode.
func NewListNode() *ListNode {
	return &ListNode{NodeType: NodeList}
}

// Append appens a child node to the list.
func (n *ListNode) Append(node Node) *ListNode {
	n.Body = append(n.Body, node)
	return n
}

func (n *ListNode) Validate() error {
	switch {
	case n.NodeType != NodeList:
		return fmt.Errorf("List Node uses an invalid type")
	case len(n.Body) == 0:
		return fmt.Errorf("List Node body is empty")
	default:
		return nil
	}
}
