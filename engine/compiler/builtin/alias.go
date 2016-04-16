package builtin

import (
	"fmt"

	"github.com/drone/drone/engine/compiler/parse"
)

type aliasOp struct {
	visitor
	index  map[string]string
	prefix string
	suffix int
}

func NewAliasOp(prefix string) Visitor {
	return &aliasOp{
		index: map[string]string{},
		prefix: prefix,
	}
}

func (v *aliasOp) VisitContainer(node *parse.ContainerNode) error {
	v.suffix++

	node.Container.Alias = node.Container.Name
	node.Container.Name = fmt.Sprintf("%s_%d", v.prefix, v.suffix)
	return nil
}
