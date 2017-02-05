package parse

// Node is an element in the parse tree.
type Node interface {
	node()
}

// empty string node
var empty = new(TextNode)

// a template is represented by a tree consisting of one
// or more of the following nodes.
type (
	// TextNode represents a string of text.
	TextNode struct {
		Value string
	}

	// FuncNode represents a string function.
	FuncNode struct {
		Param string
		Name  string
		Args  []Node
	}

	// ListNode represents a list of nodes.
	ListNode struct {
		Nodes []Node
	}

	// ParamNode struct{
	// 	Name string
	// }
	//
	// CaseNode struct {
	// 	Name string
	// 	First bool
	// }
	//
	// LowerNode struct {
	// 	Name string
	// 	First bool
	// }
	//
	// SubstrNode struct {
	// 	Name string
	// 	Pos Node
	// 	Len Node
	// }
	//
	// ReplaceNode struct {
	// 	Name string
	// 	Substring Node
	// 	Replacement Node
	// }
	//
	// TrimNode struct{
	//
	// }
	//
	// DefaultNode struct {
	// 	Name string
	// 	Default Node
	// }
)

// newTextNode returns a new TextNode.
func newTextNode(text string) *TextNode {
	return &TextNode{Value: text}
}

// newListNode returns a new ListNode.
func newListNode(nodes ...Node) *ListNode {
	return &ListNode{Nodes: nodes}
}

// newFuncNode returns a new FuncNode.
func newFuncNode(name string) *FuncNode {
	return &FuncNode{Param: name}
}

// node() defines the node in a parse tree

func (*TextNode) node() {}
func (*ListNode) node() {}
func (*FuncNode) node() {}
