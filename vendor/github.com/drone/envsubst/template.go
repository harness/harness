package envsubst

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/drone/envsubst/parse"
)

// state represents the state of template execution. It is not part of the
// template so that multiple executions can run in parallel.
type state struct {
	template *Template
	writer   io.Writer
	node     parse.Node // current node

	// maps variable names to values
	mapper func(string) string
}

// Template is the representation of a parsed shell format string.
type Template struct {
	tree *parse.Tree
}

// Parse creates a new shell format template and parses the template
// definition from string s.
func Parse(s string) (t *Template, err error) {
	t = new(Template)
	t.tree, err = parse.Parse(s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ParseFile creates a new shell format template and parses the template
// definition from the named file.
func ParseFile(path string) (*Template, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(b))
}

// Execute applies a parsed template to the specified data mapping.
func (t *Template) Execute(mapping func(string) string) (str string, err error) {
	b := new(bytes.Buffer)
	s := new(state)
	s.node = t.tree.Root
	s.mapper = mapping
	s.writer = b
	err = t.eval(s)
	if err != nil {
		return
	}
	return b.String(), nil
}

func (t *Template) eval(s *state) (err error) {
	switch node := s.node.(type) {
	case *parse.TextNode:
		err = t.evalText(s, node)
	case *parse.FuncNode:
		err = t.evalFunc(s, node)
	case *parse.ListNode:
		err = t.evalList(s, node)
	}
	return err
}

func (t *Template) evalText(s *state, node *parse.TextNode) error {
	_, err := io.WriteString(s.writer, node.Value)
	return err
}

func (t *Template) evalList(s *state, node *parse.ListNode) (err error) {
	for _, n := range node.Nodes {
		s.node = n
		err = t.eval(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) evalFunc(s *state, node *parse.FuncNode) error {
	var w = s.writer
	var buf bytes.Buffer
	var args []string
	for _, n := range node.Args {
		buf.Reset()
		s.writer = &buf
		s.node = n
		err := t.eval(s)
		if err != nil {
			return err
		}
		args = append(args, buf.String())
	}

	// restore the origin writer
	s.writer = w
	s.node = node

	v := s.mapper(node.Param)

	fn := lookupFunc(node.Name, len(args))

	_, err := io.WriteString(s.writer, fn(v, args...))
	return err
}

// lookupFunc returns the parameters substitution function by name. If the
// named function does not exists, a default function is returned.
func lookupFunc(name string, args int) substituteFunc {
	switch name {
	case ",":
		return toLowerFirst
	case ",,":
		return toLower
	case "^":
		return toUpperFirst
	case "^^":
		return toUpper
	case "#":
		if args == 0 {
			return toLen
		}
		return trimShortestPrefix
	case "##":
		return trimLongestPrefix
	case "%":
		return trimShortestSuffix
	case "%%":
		return trimLongestSuffix
	case ":":
		return toSubstr
	case "/#":
		return replacePrefix
	case "/%":
		return replaceSuffix
	case "/":
		return replaceFirst
	case "//":
		return replaceAll
	case "=", ":=", ":-":
		return toDefault
	case ":?", ":+", "-", "+":
		return toDefault
	default:
		return toDefault
	}
}
