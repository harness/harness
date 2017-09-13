package parse

// Node is an element in the parse tree.
type Node interface {
	node()
}

// ValExpr defines a value expression.
type ValExpr interface {
	Node
	value()
}

// BoolExpr defines a boolean expression.
type BoolExpr interface {
	Node
	bool()
}

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
//
type (
	// ComparisonExpr represents a two-value comparison expression.
	ComparisonExpr struct {
		Operator    Operator
		Left, Right ValExpr
	}

	// AndExpr represents an AND expression.
	AndExpr struct {
		Left, Right BoolExpr
	}

	// OrExpr represents an OR expression.
	OrExpr struct {
		Left, Right BoolExpr
	}

	// NotExpr represents a NOT expression.
	NotExpr struct {
		Expr BoolExpr
	}

	// ParenBoolExpr represents a parenthesized boolean expression.
	ParenBoolExpr struct {
		Expr BoolExpr
	}

	// BasicLit represents a basic literal.
	BasicLit struct {
		Kind  Literal // INT, REAL, TEXT
		Value []byte
	}

	// ArrayLit represents an array literal.
	ArrayLit struct {
		Values []ValExpr
	}

	// Field represents a value lookup by name.
	Field struct {
		Name []byte
	}
)

// Operator identifies the type of operator.
type Operator int

// Comparison operators.
const (
	OperatorEq Operator = iota
	OperatorLt
	OperatorLte
	OperatorGt
	OperatorGte
	OperatorNeq
	OperatorIn
	OperatorRe
	OperatorGlob
	OperatorNotIn
	OperatorNotRe
	OperatorNotGlob
)

// Literal identifies the type of literal.
type Literal int

// The list of possible literal kinds.
const (
	LiteralBool Literal = iota
	LiteralInt
	LiteralReal
	LiteralText
)

// node() defines the node in a parse tree
func (x *ComparisonExpr) node() {}
func (x *AndExpr) node()        {}
func (x *OrExpr) node()         {}
func (x *NotExpr) node()        {}
func (x *ParenBoolExpr) node()  {}
func (x *BasicLit) node()       {}
func (x *ArrayLit) node()       {}
func (x *Field) node()          {}

// bool() defines the node as a boolean expression.
func (x *ComparisonExpr) bool() {}
func (x *AndExpr) bool()        {}
func (x *OrExpr) bool()         {}
func (x *NotExpr) bool()        {}
func (x *ParenBoolExpr) bool()  {}

// value() defines the node as a value expression.
func (x *BasicLit) value() {}
func (x *ArrayLit) value() {}
func (x *Field) value()    {}
