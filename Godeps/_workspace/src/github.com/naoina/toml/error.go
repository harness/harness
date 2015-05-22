package toml

import (
	"fmt"
	"reflect"
)

func (e *parseError) Line() int {
	tokens := e.p.tokenTree.Error()
	positions := make([]int, len(tokens)*2)
	p := 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	for _, t := range translatePositions(e.p.Buffer, positions) {
		if e.p.line < t.line {
			e.p.line = t.line
		}
	}
	return e.p.line
}

type errorOutOfRange struct {
	kind reflect.Kind
	v    interface{}
}

func (err *errorOutOfRange) Error() string {
	return fmt.Sprintf("value %d is out of range for `%v` type", err.v, err.kind)
}
