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
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/go-swagger/go-swagger/spec"
)

type responseTypable struct {
	in       string
	header   *spec.Header
	response *spec.Response
}

func (ht responseTypable) Level() int { return 0 }

func (ht responseTypable) Typed(tpe, format string) {
	ht.header.Typed(tpe, format)
}

func bodyTypable(in string, schema *spec.Schema) (swaggerTypable, *spec.Schema) {
	if in == "body" {
		// get the schema for items on the schema property
		if schema == nil {
			schema = new(spec.Schema)
		}
		if schema.Items == nil {
			schema.Items = new(spec.SchemaOrArray)
		}
		if schema.Items.Schema == nil {
			schema.Items.Schema = new(spec.Schema)
		}
		schema.Typed("array", "")
		return schemaTypable{schema.Items.Schema, 0}, schema
	}
	return nil, nil
}

func (ht responseTypable) Items() swaggerTypable {
	bdt, schema := bodyTypable(ht.in, ht.response.Schema)
	if bdt != nil {
		ht.response.Schema = schema
		return bdt
	}

	if ht.header.Items == nil {
		ht.header.Items = new(spec.Items)
	}
	ht.header.Type = "array"
	return itemsTypable{ht.header.Items, 1}
}

func (ht responseTypable) SetRef(ref spec.Ref) {
	// having trouble seeing the usefulness of this one here
	ht.Schema().Ref = ref
}

func (ht responseTypable) Schema() *spec.Schema {
	if ht.response.Schema == nil {
		ht.response.Schema = new(spec.Schema)
	}
	return ht.response.Schema
}

func (ht responseTypable) SetSchema(schema *spec.Schema) {
	ht.response.Schema = schema
}
func (ht responseTypable) CollectionOf(items *spec.Items, format string) {
	ht.header.CollectionOf(items, format)
}

type headerValidations struct {
	current *spec.Header
}

func (sv headerValidations) SetMaximum(val float64, exclusive bool) {
	sv.current.Maximum = &val
	sv.current.ExclusiveMaximum = exclusive
}
func (sv headerValidations) SetMinimum(val float64, exclusive bool) {
	sv.current.Minimum = &val
	sv.current.ExclusiveMinimum = exclusive
}
func (sv headerValidations) SetMultipleOf(val float64)      { sv.current.MultipleOf = &val }
func (sv headerValidations) SetMinItems(val int64)          { sv.current.MinItems = &val }
func (sv headerValidations) SetMaxItems(val int64)          { sv.current.MaxItems = &val }
func (sv headerValidations) SetMinLength(val int64)         { sv.current.MinLength = &val }
func (sv headerValidations) SetMaxLength(val int64)         { sv.current.MaxLength = &val }
func (sv headerValidations) SetPattern(val string)          { sv.current.Pattern = val }
func (sv headerValidations) SetUnique(val bool)             { sv.current.UniqueItems = val }
func (sv headerValidations) SetCollectionFormat(val string) { sv.current.CollectionFormat = val }

func newResponseDecl(file *ast.File, decl *ast.GenDecl, ts *ast.TypeSpec) responseDecl {
	var rd responseDecl
	rd.File = file
	rd.Decl = decl
	rd.TypeSpec = ts
	rd.inferNames()
	return rd
}

type responseDecl struct {
	File      *ast.File
	Decl      *ast.GenDecl
	TypeSpec  *ast.TypeSpec
	GoName    string
	Name      string
	annotated bool
}

func (sd *responseDecl) hasAnnotation() bool {
	sd.inferNames()
	return sd.annotated
}

func (sd *responseDecl) inferNames() (goName string, name string) {
	if sd.GoName != "" {
		goName, name = sd.GoName, sd.Name
		return
	}
	goName = sd.TypeSpec.Name.Name
	name = goName
	if sd.Decl.Doc != nil {
	DECLS:
		for _, cmt := range sd.Decl.Doc.List {
			for _, ln := range strings.Split(cmt.Text, "\n") {
				matches := rxResponseOverride.FindStringSubmatch(ln)
				if len(matches) > 0 {
					sd.annotated = true
				}
				if len(matches) > 1 && len(matches[1]) > 0 {
					name = matches[1]
					break DECLS
				}
			}
		}
	}
	sd.GoName = goName
	sd.Name = name
	return
}

func newResponseParser(prog *loader.Program) *responseParser {
	return &responseParser{prog, nil, newSchemaParser(prog)}
}

type responseParser struct {
	program   *loader.Program
	postDecls []schemaDecl
	scp       *schemaParser
}

func (rp *responseParser) Parse(gofile *ast.File, target interface{}) error {
	tgt := target.(map[string]spec.Response)
	for _, decl := range gofile.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spc := range gd.Specs {
			if ts, ok := spc.(*ast.TypeSpec); ok {
				sd := newResponseDecl(gofile, gd, ts)
				if sd.hasAnnotation() {
					if err := rp.parseDecl(tgt, sd); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (rp *responseParser) parseDecl(responses map[string]spec.Response, decl responseDecl) error {
	// check if there is a swagger:parameters tag that is followed by one or more words,
	// these words are the ids of the operations this parameter struct applies to
	// once type name is found convert it to a schema, by looking up the schema in the
	// parameters dictionary that got passed into this parse method
	response := responses[decl.Name]
	resPtr := &response

	// analyze doc comment for the model
	sp := new(sectionedParser)
	sp.setDescription = func(lines []string) { resPtr.Description = joinDropLast(lines) }
	if err := sp.Parse(decl.Decl.Doc); err != nil {
		return err
	}

	// analyze struct body for fields etc
	// each exported struct field:
	// * gets a type mapped to a go primitive
	// * perhaps gets a format
	// * has to document the validations that apply for the type and the field
	// * when the struct field points to a model it becomes a ref: #/definitions/ModelName
	// * comments that aren't tags is used as the description
	if tpe, ok := decl.TypeSpec.Type.(*ast.StructType); ok {
		if err := rp.parseStructType(decl.File, resPtr, tpe, make(map[string]struct{})); err != nil {
			return err
		}
	}

	responses[decl.Name] = response
	return nil
}

func (rp *responseParser) parseEmbeddedStruct(gofile *ast.File, response *spec.Response, expr ast.Expr, seenPreviously map[string]struct{}) error {
	switch tpe := expr.(type) {
	case *ast.Ident:
		// do lookup of type
		// take primitives into account, they should result in an error for swagger
		pkg, err := rp.scp.packageForFile(gofile)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		file, _, ts, err := findSourceFile(pkg, tpe.Name)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			return rp.parseStructType(file, response, st, seenPreviously)
		}
	case *ast.SelectorExpr:
		// look up package, file and then type
		pkg, err := rp.scp.packageForSelector(gofile, tpe.X)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		file, _, ts, err := findSourceFile(pkg, tpe.Sel.Name)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			return rp.parseStructType(file, response, st, seenPreviously)
		}
	}
	return fmt.Errorf("unable to resolve embedded struct for: %v\n", expr)
}

func (rp *responseParser) parseStructType(gofile *ast.File, response *spec.Response, tpe *ast.StructType, seenPreviously map[string]struct{}) error {
	if tpe.Fields != nil {

		seenProperties := seenPreviously

		for _, fld := range tpe.Fields.List {
			if len(fld.Names) == 0 {
				// when the embedded struct is annotated with swagger:allOf it will be used as allOf property
				// otherwise the fields will just be included as normal properties
				if err := rp.parseEmbeddedStruct(gofile, response, fld.Type, seenProperties); err != nil {
					return err
				}
			}
		}

		for _, fld := range tpe.Fields.List {
			var nm string
			if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
				nm = fld.Names[0].Name
				if fld.Tag != nil && len(strings.TrimSpace(fld.Tag.Value)) > 0 {
					tv, err := strconv.Unquote(fld.Tag.Value)
					if err != nil {
						return err
					}

					if strings.TrimSpace(tv) != "" {
						st := reflect.StructTag(tv)
						if st.Get("json") != "" {
							nm = strings.Split(st.Get("json"), ",")[0]
						}
					}
				}

				var in string
				// scan for param location first, this changes some behavior down the line
				if fld.Doc != nil {
					for _, cmt := range fld.Doc.List {
						for _, line := range strings.Split(cmt.Text, "\n") {
							matches := rxIn.FindStringSubmatch(line)
							if len(matches) > 0 && len(strings.TrimSpace(matches[1])) > 0 {
								in = strings.TrimSpace(matches[1])
							}
						}
					}
				}

				ps := response.Headers[nm]
				if err := parseProperty(rp.scp, gofile, fld.Type, responseTypable{in, &ps, response}); err != nil {
					return err
				}

				sp := new(sectionedParser)
				sp.setDescription = func(lines []string) { ps.Description = joinDropLast(lines) }
				sp.taggers = []tagParser{
					newSingleLineTagParser("maximum", &setMaximum{headerValidations{&ps}, rxf(rxMaximumFmt, "")}),
					newSingleLineTagParser("minimum", &setMinimum{headerValidations{&ps}, rxf(rxMinimumFmt, "")}),
					newSingleLineTagParser("multipleOf", &setMultipleOf{headerValidations{&ps}, rxf(rxMultipleOfFmt, "")}),
					newSingleLineTagParser("minLength", &setMinLength{headerValidations{&ps}, rxf(rxMinLengthFmt, "")}),
					newSingleLineTagParser("maxLength", &setMaxLength{headerValidations{&ps}, rxf(rxMaxLengthFmt, "")}),
					newSingleLineTagParser("pattern", &setPattern{headerValidations{&ps}, rxf(rxPatternFmt, "")}),
					newSingleLineTagParser("collectionFormat", &setCollectionFormat{headerValidations{&ps}, rxf(rxCollectionFormatFmt, "")}),
					newSingleLineTagParser("minItems", &setMinItems{headerValidations{&ps}, rxf(rxMinItemsFmt, "")}),
					newSingleLineTagParser("maxItems", &setMaxItems{headerValidations{&ps}, rxf(rxMaxItemsFmt, "")}),
					newSingleLineTagParser("unique", &setUnique{headerValidations{&ps}, rxf(rxUniqueFmt, "")}),
				}
				itemsTaggers := func(items *spec.Items, level int) []tagParser {
					// the expression is 1-index based not 0-index
					itemsPrefix := fmt.Sprintf(rxItemsPrefixFmt, level+1)

					return []tagParser{
						newSingleLineTagParser(fmt.Sprintf("items%dMaximum", level), &setMaximum{itemsValidations{items}, rxf(rxMaximumFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMinimum", level), &setMinimum{itemsValidations{items}, rxf(rxMinimumFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMultipleOf", level), &setMultipleOf{itemsValidations{items}, rxf(rxMultipleOfFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMinLength", level), &setMinLength{itemsValidations{items}, rxf(rxMinLengthFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMaxLength", level), &setMaxLength{itemsValidations{items}, rxf(rxMaxLengthFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dPattern", level), &setPattern{itemsValidations{items}, rxf(rxPatternFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dCollectionFormat", level), &setCollectionFormat{itemsValidations{items}, rxf(rxCollectionFormatFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMinItems", level), &setMinItems{itemsValidations{items}, rxf(rxMinItemsFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dMaxItems", level), &setMaxItems{itemsValidations{items}, rxf(rxMaxItemsFmt, itemsPrefix)}),
						newSingleLineTagParser(fmt.Sprintf("items%dUnique", level), &setUnique{itemsValidations{items}, rxf(rxUniqueFmt, itemsPrefix)}),
					}
				}

				// check if this is a primitive, if so parse the validations from the
				// doc comments of the slice declaration.
				if ftped, ok := fld.Type.(*ast.ArrayType); ok {
					ftpe := ftped
					items, level := ps.Items, 0
					for items != nil {
						switch iftpe := ftpe.Elt.(type) {
						case *ast.ArrayType:
							eleTaggers := itemsTaggers(items, level)
							sp.taggers = append(eleTaggers, sp.taggers...)
							ftpe = iftpe
						case *ast.Ident:
							if iftpe.Obj == nil {
								sp.taggers = append(itemsTaggers(items, level), sp.taggers...)
							}
							break
						default:
							return fmt.Errorf("unknown field type ele for %q", nm)
						}
						items = items.Items
						level = level + 1
					}
				}

				if err := sp.Parse(fld.Doc); err != nil {
					return err
				}

				if in != "body" {
					seenProperties[nm] = struct{}{}
					if response.Headers == nil {
						response.Headers = make(map[string]spec.Header)
					}
					response.Headers[nm] = ps
				}
			}
		}

		for k := range response.Headers {
			if _, ok := seenProperties[k]; !ok {
				delete(response.Headers, k)
			}
		}
	}

	return nil
}
