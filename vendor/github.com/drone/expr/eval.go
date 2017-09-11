package expr

import (
	"bytes"
	"path/filepath"
	"regexp"

	"github.com/drone/expr/parse"
)

// state represents the state of an execution. It's not part of the
// statement so that multiple executions of the same statement
// can execute in parallel.
type state struct {
	node parse.Node
	vars Row
}

// at marks the state to be on node n, for error reporting.
func (s *state) at(node parse.Node) {
	s.node = node
}

// Walk functions step through the major pieces of the template structure,
// generating output as they go.
func (s *state) walk(node parse.BoolExpr) bool {
	s.at(node)

	switch node := node.(type) {
	case *parse.ComparisonExpr:
		return s.eval(node)
	case *parse.AndExpr:
		return s.walk(node.Left) && s.walk(node.Right)
	case *parse.OrExpr:
		return s.walk(node.Left) || s.walk(node.Right)
	case *parse.NotExpr:
		return !s.walk(node.Expr)
	case *parse.ParenBoolExpr:
		return s.walk(node.Expr)
	default:
		panic("invalid node type")
	}
}

func (s *state) eval(node *parse.ComparisonExpr) bool {
	switch node.Operator {
	case parse.OperatorEq:
		return s.evalEq(node)
	case parse.OperatorGt:
		return s.evalGt(node)
	case parse.OperatorGte:
		return s.evalGte(node)
	case parse.OperatorLt:
		return s.evalLt(node)
	case parse.OperatorLte:
		return s.evalLte(node)
	case parse.OperatorNeq:
		return !s.evalEq(node)
	case parse.OperatorGlob:
		return s.evalGlob(node)
	case parse.OperatorNotGlob:
		return !s.evalGlob(node)
	case parse.OperatorRe:
		return s.evalRegexp(node)
	case parse.OperatorNotRe:
		return !s.evalRegexp(node)
	case parse.OperatorIn:
		return s.evalIn(node)
	case parse.OperatorNotIn:
		return !s.evalIn(node)
	default:
		panic("inalid operator type")
	}
}

func (s *state) evalEq(node *parse.ComparisonExpr) bool {
	return bytes.Equal(
		s.toValue(node.Left),
		s.toValue(node.Right),
	)
}

func (s *state) evalGt(node *parse.ComparisonExpr) bool {
	return bytes.Compare(
		s.toValue(node.Left),
		s.toValue(node.Right),
	) == 1
}

func (s *state) evalGte(node *parse.ComparisonExpr) bool {
	return bytes.Compare(
		s.toValue(node.Left),
		s.toValue(node.Right),
	) >= 0
}

func (s *state) evalLt(node *parse.ComparisonExpr) bool {
	return bytes.Compare(
		s.toValue(node.Left),
		s.toValue(node.Right),
	) == -1
}

func (s *state) evalLte(node *parse.ComparisonExpr) bool {
	return bytes.Compare(
		s.toValue(node.Left),
		s.toValue(node.Right),
	) <= 0
}

func (s *state) evalGlob(node *parse.ComparisonExpr) bool {
	match, _ := filepath.Match(
		string(s.toValue(node.Right)),
		string(s.toValue(node.Left)),
	)
	return match
}

func (s *state) evalRegexp(node *parse.ComparisonExpr) bool {
	match, _ := regexp.Match(
		string(s.toValue(node.Right)),
		s.toValue(node.Left),
	)
	return match
}

func (s *state) evalIn(node *parse.ComparisonExpr) bool {
	left := s.toValue(node.Left)
	right, ok := node.Right.(*parse.ArrayLit)
	if !ok {
		panic("expected array literal")
	}

	for _, expr := range right.Values {
		if bytes.Equal(left, s.toValue(expr)) {
			return true
		}
	}
	return false
}

func (s *state) toValue(expr parse.ValExpr) []byte {
	switch node := expr.(type) {
	case *parse.Field:
		return s.vars.Field(node.Name)
	case *parse.BasicLit:
		return node.Value
	default:
		panic("invalid expression type")
	}
}

// errRecover is the handler that turns panics into returns.
func errRecover(err *error) {
	if e := recover(); e != nil {
		*err = e.(error)
	}
}
