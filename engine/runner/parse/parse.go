package parse

import "encoding/json"

// Tree is the intermediate representation of a pipeline.
type Tree struct {
	*ListNode // top-level Tree node
}

// New allocates a new Tree.
func NewTree() *Tree {
	return &Tree{
		NewListNode(),
	}
}

// Parse parses a JSON encoded Tree.
func Parse(data []byte) (*Tree, error) {
	tree := &Tree{}
	err := tree.UnmarshalJSON(data)
	return tree, err
}

// MarshalJSON implements the Marshaler interface and returns
// a JSON encoded representation of the Tree.
func (t *Tree) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ListNode)
}

// UnmarshalJSON implements the Unmarshaler interface and returns
// a Tree from a JSON representation.
func (t *Tree) UnmarshalJSON(data []byte) error {
	block, err := decodeList(data)
	if err != nil {
		return nil
	}
	t.ListNode = block.(*ListNode)
	return nil
}

//
// below are custom decoding functions. We cannot use the default json
// decoder because the tree structure uses interfaces and the json decoder
// has difficulty ascertaining the interface type when decoding.
//

func decodeNode(data []byte) (Node, error) {
	node := &nodeType{}

	err := json.Unmarshal(data, node)
	if err != nil {
		return nil, err
	}
	switch node.Type {
	case NodeList:
		return decodeList(data)
	case NodeDefer:
		return decodeDefer(data)
	case NodeError:
		return decodeError(data)
	case NodeRecover:
		return decodeRecover(data)
	case NodeParallel:
		return decodeParallel(data)
	case NodeRun:
		return decodeRun(data)
	}
	return nil, nil
}

func decodeNodes(data []json.RawMessage) ([]Node, error) {
	var nodes []Node
	for _, d := range data {
		node, err := decodeNode(d)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func decodeList(data []byte) (Node, error) {
	v := &nodeList{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	b, err := decodeNodes(v.Body)
	if err != nil {
		return nil, err
	}
	n := NewListNode()
	n.Body = b
	return n, nil
}

func decodeDefer(data []byte) (Node, error) {
	v := &nodeDefer{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	b, err := decodeNode(v.Body)
	if err != nil {
		return nil, err
	}
	d, err := decodeNode(v.Defer)
	if err != nil {
		return nil, err
	}
	n := NewDeferNode()
	n.Body = b
	n.Defer = d
	return n, nil
}

func decodeError(data []byte) (Node, error) {
	v := &nodeError{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	b, err := decodeNode(v.Body)
	if err != nil {
		return nil, err
	}
	d, err := decodeNode(v.Defer)
	if err != nil {
		return nil, err
	}
	n := NewErrorNode()
	n.Body = b
	n.Defer = d
	return n, nil
}

func decodeRecover(data []byte) (Node, error) {
	v := &nodeRecover{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	b, err := decodeNode(v.Body)
	if err != nil {
		return nil, err
	}
	n := NewRecoverNode()
	n.Body = b
	return n, nil
}

func decodeParallel(data []byte) (Node, error) {
	v := &nodeParallel{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	b, err := decodeNodes(v.Body)
	if err != nil {
		return nil, err
	}
	n := NewParallelNode()
	n.Body = b
	n.Limit = v.Limit
	return n, nil
}

func decodeRun(data []byte) (Node, error) {
	v := &nodeRun{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	return &RunNode{NodeRun, v.Name, v.Detach, v.Silent}, nil
}

//
// below are intermediate representations of the node structures
// since we cannot simply encode / decode using the built-in json
// encoding and decoder.
//

type nodeType struct {
	Type NodeType `json:"type"`
}

type nodeDefer struct {
	Type  NodeType        `json:"type"`
	Body  json.RawMessage `json:"body"`
	Defer json.RawMessage `json:"defer"`
}

type nodeError struct {
	Type  NodeType        `json:"type"`
	Body  json.RawMessage `json:"body"`
	Defer json.RawMessage `json:"defer"`
}

type nodeList struct {
	Type NodeType          `json:"type"`
	Body []json.RawMessage `json:"body"`
}

type nodeRecover struct {
	Type NodeType        `json:"type"`
	Body json.RawMessage `json:"body"`
}

type nodeParallel struct {
	Type  NodeType          `json:"type"`
	Body  []json.RawMessage `json:"body"`
	Limit int               `json:"limit"`
}

type nodeRun struct {
	Type   NodeType `json:"type"`
	Name   string   `json:"name"`
	Detach bool     `json:"detach,omitempty"`
	Silent bool     `json:"silent,omitempty"`
}
