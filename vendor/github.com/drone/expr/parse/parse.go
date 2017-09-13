package parse

import (
	"bytes"
	"fmt"
)

// Tree is the representation of a single parsed SQL statement.
type Tree struct {
	Root BoolExpr

	// Parsing only; cleared after parse.
	lex *lexer
}

// Parse parses the SQL statement and returns a Tree.
func Parse(buf []byte) (*Tree, error) {
	t := new(Tree)
	t.lex = new(lexer)
	return t.Parse(buf)
}

// Parse parses the SQL statement buffer to construct an ast
// representation for execution.
func (t *Tree) Parse(buf []byte) (tree *Tree, err error) {
	defer t.recover(&err)
	t.lex.init(buf)
	t.Root = t.parseExpr()
	return t, nil
}

// recover is the handler that turns panics into returns.
func (t *Tree) recover(err *error) {
	if e := recover(); e != nil {
		*err = e.(error)
	}
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("selector: parse error:%d: %s", t.lex.start, format)
	panic(fmt.Errorf(format, args...))
}

func (t *Tree) parseExpr() BoolExpr {
	if t.lex.peek() == tokenNot {
		t.lex.scan()
		return t.parseNot()
	}

	left := t.parseVal()
	node := t.parseComparison(left)

	switch t.lex.scan() {
	case tokenOr:
		return t.parseOr(node)
	case tokenAnd:
		return t.parseAnd(node)
	default:
		return node
	}
}

func (t *Tree) parseAnd(left BoolExpr) BoolExpr {
	node := new(AndExpr)
	node.Left = left
	node.Right = t.parseExpr()
	return node
}

func (t *Tree) parseOr(left BoolExpr) BoolExpr {
	node := new(OrExpr)
	node.Left = left
	node.Right = t.parseExpr()
	return node
}

func (t *Tree) parseNot() BoolExpr {
	node := new(NotExpr)
	node.Expr = t.parseExpr()
	return node
}

func (t *Tree) parseComparison(left ValExpr) BoolExpr {
	var negate bool
	if t.lex.peek() == tokenNot {
		t.lex.scan()
		negate = true
	}

	node := new(ComparisonExpr)
	node.Operator = t.parseOperator()
	node.Left = left

	if negate {
		switch node.Operator {
		case OperatorIn:
			node.Operator = OperatorNotIn
		case OperatorGlob:
			node.Operator = OperatorNotGlob
		case OperatorRe:
			node.Operator = OperatorNotRe
		}
	}

	switch node.Operator {
	case OperatorIn, OperatorNotIn:
		node.Right = t.parseList()
	case OperatorRe, OperatorNotRe:
		// TODO placeholder for custom Regexp Node
		node.Right = t.parseVal()
	default:
		node.Right = t.parseVal()
	}
	return node
}

func (t *Tree) parseOperator() (op Operator) {
	switch t.lex.scan() {
	case tokenEq:
		return OperatorEq
	case tokenGt:
		return OperatorGt
	case tokenGte:
		return OperatorGte
	case tokenLt:
		return OperatorLt
	case tokenLte:
		return OperatorLte
	case tokenNeq:
		return OperatorNeq
	case tokenIn:
		return OperatorIn
	case tokenRegexp:
		return OperatorRe
	case tokenGlob:
		return OperatorGlob
	default:
		t.errorf("illegal operator")
		return
	}
}

func (t *Tree) parseVal() ValExpr {
	switch t.lex.scan() {
	case tokenIdent:
		node := new(Field)
		node.Name = t.lex.bytes()
		return node
	case tokenText:
		return t.parseText()
	case tokenReal, tokenInteger, tokenTrue, tokenFalse:
		node := new(BasicLit)
		node.Value = t.lex.bytes()
		return node
	default:
		t.errorf("illegal value expression")
		return nil
	}
}

func (t *Tree) parseList() ValExpr {
	if t.lex.scan() != tokenLparen {
		t.errorf("unexpected token, expecting (")
		return nil
	}
	node := new(ArrayLit)
	for {
		next := t.lex.peek()
		switch next {
		case tokenEOF:
			t.errorf("unexpected eof, expecting )")
		case tokenComma:
			t.lex.scan()
		case tokenRparen:
			t.lex.scan()
			return node
		default:
			child := t.parseVal()
			node.Values = append(node.Values, child)
		}
	}
}

func (t *Tree) parseText() ValExpr {
	node := new(BasicLit)
	node.Value = t.lex.bytes()

	// this is where we strip the starting and ending quote
	// and unescape the string. On the surface this might look
	// like it is subject to index out of bounds errors but
	// it is safe because it is already verified by the lexer.
	node.Value = node.Value[1 : len(node.Value)-1]
	node.Value = bytes.Replace(node.Value, quoteEscaped, quoteUnescaped, -1)
	return node
}

// errString indicates the string literal does no have the right syntax.
// var errString = errors.New("invalid string literal")

var (
	quoteEscaped   = []byte("\\'")
	quoteUnescaped = []byte("'")
)

// unquote interprets buf as a single-quoted literal, returning the
// value that buf quotes.
// func unquote(buf []byte) ([]byte, error) {
// 	n := len(buf)
// 	if n < 2 {
// 		return nil, errString
// 	}
// 	quote := buf[0]
// 	if quote != quoteUnescaped[0] {
// 		return nil, errString
// 	}
// 	if quote != buf[n-1] {
// 		return nil, errString
// 	}
// 	buf = buf[1 : n-1]
// 	return bytes.Replace(buf, quoteEscaped, quoteUnescaped, -1), nil
// }
