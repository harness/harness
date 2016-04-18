// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scan

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/go-swagger/go-swagger/spec"

	"golang.org/x/tools/go/loader"
)

func opConsumesSetter(op *spec.Operation) func([]string) {
	return func(consumes []string) { op.Consumes = consumes }
}

func opProducesSetter(op *spec.Operation) func([]string) {
	return func(produces []string) { op.Produces = produces }
}

func opSchemeSetter(op *spec.Operation) func([]string) {
	return func(schemes []string) { op.Schemes = schemes }
}

func opSecurityDefsSetter(op *spec.Operation) func([]map[string][]string) {
	return func(securityDefs []map[string][]string) { op.Security = securityDefs }
}

func opResponsesSetter(op *spec.Operation) func(*spec.Response, map[int]spec.Response) {
	return func(def *spec.Response, scr map[int]spec.Response) {
		if op.Responses == nil {
			op.Responses = new(spec.Responses)
		}
		op.Responses.Default = def
		op.Responses.StatusCodeResponses = scr
	}
}

func newRoutesParser(prog *loader.Program) *routesParser {
	return &routesParser{
		program: prog,
	}
}

type routesParser struct {
	program     *loader.Program
	definitions map[string]spec.Schema
	operations  map[string]*spec.Operation
	responses   map[string]spec.Response
}

func (rp *routesParser) Parse(gofile *ast.File, target interface{}) error {
	tgt := target.(*spec.Paths)
	for _, comsec := range gofile.Comments {

		// check if this is a route comment section
		var method, path, id string
		var tags []string
		var remaining *ast.CommentGroup
		var justMatched bool

		for _, cmt := range comsec.List {
			for _, line := range strings.Split(cmt.Text, "\n") {
				matches := rxRoute.FindStringSubmatch(line)
				if len(matches) > 3 {
					method, path, id = matches[1], matches[2], matches[len(matches)-1]
					tags = rxSpace.Split(matches[3], -1)
					if len(matches[3]) == 0 {
						tags = nil
					}
					justMatched = true
				} else if method != "" {
					if remaining == nil {
						remaining = new(ast.CommentGroup)
					}
					if !justMatched || strings.TrimSpace(rxStripComments.ReplaceAllString(line, "")) != "" {
						cc := new(ast.Comment)
						cc.Slash = cmt.Slash
						cc.Text = line
						remaining.List = append(remaining.List, cc)
						justMatched = false
					}
				}
			}
		}

		if method == "" {
			continue // it's not, next!
		}

		pthObj := tgt.Paths[path]
		op := rp.operations[id]
		if op == nil {
			op = new(spec.Operation)
			op.ID = id
		}
		switch strings.ToUpper(method) {
		case "GET":
			if pthObj.Get != nil {
				if id == pthObj.Get.ID {
					op = pthObj.Get
				} else {
					pthObj.Get = op
				}
			} else {
				pthObj.Get = op
			}

		case "POST":
			if pthObj.Post != nil {
				if id == pthObj.Post.ID {
					op = pthObj.Post
				} else {
					pthObj.Post = op
				}
			} else {
				pthObj.Post = op
			}

		case "PUT":
			if pthObj.Put != nil {
				if id == pthObj.Put.ID {
					op = pthObj.Put
				} else {
					pthObj.Put = op
				}
			} else {
				pthObj.Put = op
			}

		case "PATCH":
			if pthObj.Patch != nil {
				if id == pthObj.Patch.ID {
					op = pthObj.Patch
				} else {
					pthObj.Patch = op
				}
			} else {
				pthObj.Patch = op
			}

		case "HEAD":
			if pthObj.Head != nil {
				if id == pthObj.Head.ID {
					op = pthObj.Head
				} else {
					pthObj.Head = op
				}
			} else {
				pthObj.Head = op
			}

		case "DELETE":
			if pthObj.Delete != nil {
				if id == pthObj.Delete.ID {
					op = pthObj.Delete
				} else {
					pthObj.Delete = op
				}
			} else {
				pthObj.Delete = op
			}

		case "OPTIONS":
			if pthObj.Options != nil {
				if id == pthObj.Options.ID {
					op = pthObj.Options
				} else {
					pthObj.Options = op
				}
			} else {
				pthObj.Options = op
			}
		}
		op.Tags = tags
		sp := new(sectionedParser)
		sp.setTitle = func(lines []string) { op.Summary = joinDropLast(lines) }
		sp.setDescription = func(lines []string) { op.Description = joinDropLast(lines) }
		sr := newSetResponses(rp.definitions, rp.responses, opResponsesSetter(op))
		sp.taggers = []tagParser{
			newMultiLineTagParser("Consumes", newMultilineDropEmptyParser(rxConsumes, opConsumesSetter(op))),
			newMultiLineTagParser("Produces", newMultilineDropEmptyParser(rxProduces, opProducesSetter(op))),
			newSingleLineTagParser("Schemes", newSetSchemes(opSchemeSetter(op))),
			newMultiLineTagParser("Security", newSetSecurityDefinitions(rxSecuritySchemes, opSecurityDefsSetter(op))),
			newMultiLineTagParser("Responses", sr),
		}
		if err := sp.Parse(remaining); err != nil {
			return fmt.Errorf("operation (%s): %v", op.ID, err)
		}

		if tgt.Paths == nil {
			tgt.Paths = make(map[string]spec.PathItem)
		}
		tgt.Paths[path] = pthObj
	}

	return nil
}
