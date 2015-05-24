package toml

import (
	"fmt"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/toml/ast"
)

// Parse returns an AST representation of TOML.
// The toplevel is represented by a table.
func Parse(data []byte) (*ast.Table, error) {
	d := &parseState{p: &tomlParser{Buffer: string(data)}}
	d.init()

	if err := d.parse(); err != nil {
		return nil, err
	}

	return d.p.toml.table, nil
}

type parseState struct {
	p *tomlParser
}

func (d *parseState) init() {
	d.p.Init()
	d.p.toml.init()
}

func (d *parseState) parse() error {
	if err := d.p.Parse(); err != nil {
		if err, ok := err.(*parseError); ok {
			return fmt.Errorf("toml: line %d: parse error", err.Line())
		}
		return err
	}
	return d.execute()
}

func (d *parseState) execute() (err error) {
	defer func() {
		e := recover()
		if e != nil {
			cerr, ok := e.(convertError)
			if !ok {
				panic(e)
			}
			err = cerr.err
		}
	}()
	d.p.Execute()
	return nil
}
