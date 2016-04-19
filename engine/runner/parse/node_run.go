package parse

import "fmt"

type RunNode struct {
	NodeType `json:"type"`

	Name   string `json:"name"`
	Detach bool   `json:"detach,omitempty"`
	Silent bool   `json:"silent,omitempty"`
}

func (n *RunNode) SetName(name string) *RunNode {
	n.Name = name
	return n
}

func (n *RunNode) SetDetach(detach bool) *RunNode {
	n.Detach = detach
	return n
}

func (n *RunNode) SetSilent(silent bool) *RunNode {
	n.Silent = silent
	return n
}

func NewRunNode() *RunNode {
	return &RunNode{NodeType: NodeRun}
}

func (n *RunNode) Validate() error {
	switch {
	case n.NodeType != NodeRun:
		return fmt.Errorf("Run Node uses an invalid type")
	case n.Name == "":
		return fmt.Errorf("Run Node has an invalid name")
	default:
		return nil
	}
}
