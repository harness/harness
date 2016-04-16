package parse

import "fmt"

// ParallelNode executes a list of child nodes in parallel.
type ParallelNode struct {
	NodeType `json:"type"`

	Body  []Node `json:"body"`  // nodes for parallel evaluation.
	Limit int    `json:"limit"` // limit for parallel evaluation.
}

func NewParallelNode() *ParallelNode {
	return &ParallelNode{NodeType: NodeParallel}
}

func (n *ParallelNode) Append(node Node) *ParallelNode {
	n.Body = append(n.Body, node)
	return n
}

func (n *ParallelNode) SetLimit(limit int) *ParallelNode {
	n.Limit = limit
	return n
}

func (n *ParallelNode) Validate() error {
	switch {
	case n.NodeType != NodeParallel:
		return fmt.Errorf("Parallel Node uses an invalid type")
	case len(n.Body) == 0:
		return fmt.Errorf("Parallel Node body is empty")
	default:
		return nil
	}
}
