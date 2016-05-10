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
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/go-swagger/go-swagger/spec"
)

type schemaTypable struct {
	schema *spec.Schema
	level  int
}

func (st schemaTypable) Typed(tpe, format string) {
	st.schema.Typed(tpe, format)
}

func (st schemaTypable) SetRef(ref spec.Ref) {
	st.schema.Ref = ref
}

func (st schemaTypable) Schema() *spec.Schema {
	return st.schema
}

func (st schemaTypable) Items() swaggerTypable {
	if st.schema.Items == nil {
		st.schema.Items = new(spec.SchemaOrArray)
	}
	if st.schema.Items.Schema == nil {
		st.schema.Items.Schema = new(spec.Schema)
	}

	st.schema.Typed("array", "")
	return schemaTypable{st.schema.Items.Schema, st.level + 1}
}
func (st schemaTypable) Level() int { return st.level }

type schemaValidations struct {
	current *spec.Schema
}

func (sv schemaValidations) SetMaximum(val float64, exclusive bool) {
	sv.current.Maximum = &val
	sv.current.ExclusiveMaximum = exclusive
}
func (sv schemaValidations) SetMinimum(val float64, exclusive bool) {
	sv.current.Minimum = &val
	sv.current.ExclusiveMinimum = exclusive
}
func (sv schemaValidations) SetMultipleOf(val float64) { sv.current.MultipleOf = &val }
func (sv schemaValidations) SetMinItems(val int64)     { sv.current.MinItems = &val }
func (sv schemaValidations) SetMaxItems(val int64)     { sv.current.MaxItems = &val }
func (sv schemaValidations) SetMinLength(val int64)    { sv.current.MinLength = &val }
func (sv schemaValidations) SetMaxLength(val int64)    { sv.current.MaxLength = &val }
func (sv schemaValidations) SetPattern(val string)     { sv.current.Pattern = val }
func (sv schemaValidations) SetUnique(val bool)        { sv.current.UniqueItems = val }

func newSchemaAnnotationParser(goName string) *schemaAnnotationParser {
	return &schemaAnnotationParser{GoName: goName, rx: rxModelOverride}
}

type schemaAnnotationParser struct {
	GoName string
	Name   string
	rx     *regexp.Regexp
}

func (sap *schemaAnnotationParser) Matches(line string) bool {
	return sap.rx.MatchString(line)
}

func (sap *schemaAnnotationParser) Parse(lines []string) error {
	if sap.Name != "" {
		return nil
	}

	if len(lines) > 0 {
		for _, line := range lines {
			matches := sap.rx.FindStringSubmatch(line)
			if len(matches) > 1 && len(matches[1]) > 0 {
				sap.Name = matches[1]
				return nil
			}
		}
	}
	return nil
}

type schemaDecl struct {
	File      *ast.File
	Decl      *ast.GenDecl
	TypeSpec  *ast.TypeSpec
	GoName    string
	Name      string
	annotated bool
}

func newSchemaDecl(file *ast.File, decl *ast.GenDecl, ts *ast.TypeSpec) *schemaDecl {
	sd := &schemaDecl{
		File:     file,
		Decl:     decl,
		TypeSpec: ts,
	}
	sd.inferNames()
	return sd
}

func (sd *schemaDecl) hasAnnotation() bool {
	sd.inferNames()
	return sd.annotated
}

func (sd *schemaDecl) inferNames() (goName string, name string) {
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
				matches := rxModelOverride.FindStringSubmatch(ln)
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

type schemaParser struct {
	program   *loader.Program
	postDecls []schemaDecl
	known     map[string]spec.Schema
}

func newSchemaParser(prog *loader.Program) *schemaParser {
	scp := new(schemaParser)
	scp.program = prog
	scp.known = make(map[string]spec.Schema)
	return scp
}

func (scp *schemaParser) Parse(gofile *ast.File, target interface{}) error {
	tgt := target.(map[string]spec.Schema)
	for _, decl := range gofile.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spc := range gd.Specs {
			if ts, ok := spc.(*ast.TypeSpec); ok {
				sd := newSchemaDecl(gofile, gd, ts)
				if err := scp.parseDecl(tgt, sd); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (scp *schemaParser) parseDecl(definitions map[string]spec.Schema, decl *schemaDecl) error {
	// check if there is a swagger:model tag that is followed by a word,
	// this word is the type name for swagger
	// the package and type are recorded in the extensions
	// once type name is found convert it to a schema, by looking up the schema in the
	// definitions dictionary that got passed into this parse method
	decl.inferNames()
	schema := definitions[decl.Name]
	schPtr := &schema

	// analyze doc comment for the model
	sp := new(sectionedParser)
	sp.setTitle = func(lines []string) { schema.Title = joinDropLast(lines) }
	sp.setDescription = func(lines []string) { schema.Description = joinDropLast(lines) }
	if err := sp.Parse(decl.Decl.Doc); err != nil {
		return err
	}

	// analyze struct body for fields etc
	// each exported struct field:
	// * gets a type mapped to a go primitive
	// * perhaps gets a format
	// * has to document the validations that apply for the type and the field
	// * when the struct field points to a model it becomes a ref: #/definitions/ModelName
	// * the first line of the comment is the title
	// * the following lines are the description
	switch tpe := decl.TypeSpec.Type.(type) {
	case *ast.StructType:
		if err := scp.parseStructType(decl.File, schPtr, tpe, make(map[string]struct{})); err != nil {
			return err
		}
	case *ast.InterfaceType:
		if err := scp.parseInterfaceType(decl.File, schPtr, tpe, make(map[string]struct{})); err != nil {
			return err
		}
	default:
	}

	if decl.Name != decl.GoName {
		schPtr.AddExtension("x-go-name", decl.GoName)
	}
	for _, pkgInfo := range scp.program.AllPackages {
		if pkgInfo.Importable {
			for _, fil := range pkgInfo.Files {
				if fil.Pos() == decl.File.Pos() {
					schPtr.AddExtension("x-go-package", pkgInfo.Pkg.Path())
				}
			}
		}
	}
	definitions[decl.Name] = schema
	return nil
}

func (scp *schemaParser) parseEmbeddedType(gofile *ast.File, schema *spec.Schema, expr ast.Expr, seenPreviously map[string]struct{}) error {
	switch tpe := expr.(type) {
	case *ast.Ident:
		// do lookup of type
		// take primitives into account, they should result in an error for swagger
		pkg, err := scp.packageForFile(gofile)
		if err != nil {
			return err
		}
		file, _, ts, err := findSourceFile(pkg, tpe.Name)
		if err != nil {
			return err
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			return scp.parseStructType(file, schema, st, seenPreviously)
		}
		if st, ok := ts.Type.(*ast.InterfaceType); ok {
			return scp.parseInterfaceType(file, schema, st, seenPreviously)
		}

	case *ast.SelectorExpr:
		// look up package, file and then type
		pkg, err := scp.packageForSelector(gofile, tpe.X)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		file, _, ts, err := findSourceFile(pkg, tpe.Sel.Name)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			return scp.parseStructType(file, schema, st, seenPreviously)
		}
		if st, ok := ts.Type.(*ast.InterfaceType); ok {
			return scp.parseInterfaceType(file, schema, st, seenPreviously)
		}
	}
	return fmt.Errorf("unable to resolve embedded struct for: %v\n", expr)
}

func (scp *schemaParser) parseAllOfMember(gofile *ast.File, schema *spec.Schema, expr ast.Expr, seenPreviously map[string]struct{}) error {
	// TODO: check if struct is annotated with swagger:model or known in the definitions otherwise
	var pkg *loader.PackageInfo
	var file *ast.File
	var gd *ast.GenDecl
	var ts *ast.TypeSpec
	var err error

	switch tpe := expr.(type) {
	case *ast.Ident:
		// do lookup of type
		// take primitives into account, they should result in an error for swagger
		pkg, err = scp.packageForFile(gofile)
		if err != nil {
			return err
		}
		file, gd, ts, err = findSourceFile(pkg, tpe.Name)
		if err != nil {
			return err
		}

	case *ast.SelectorExpr:
		// look up package, file and then type
		pkg, err = scp.packageForSelector(gofile, tpe.X)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
		file, gd, ts, err = findSourceFile(pkg, tpe.Sel.Name)
		if err != nil {
			return fmt.Errorf("embedded struct: %v", err)
		}
	default:
		return fmt.Errorf("unable to resolve allOf member for: %v\n", expr)
	}

	sd := newSchemaDecl(file, gd, ts)
	if sd.hasAnnotation() {
		ref, err := spec.NewRef("#/definitions/" + sd.Name)
		if err != nil {
			return err
		}
		schema.Ref = ref
		scp.postDecls = append(scp.postDecls, *sd)
	} else {
		switch st := ts.Type.(type) {
		case *ast.StructType:
			return scp.parseStructType(file, schema, st, seenPreviously)
		case *ast.InterfaceType:
			return scp.parseInterfaceType(file, schema, st, seenPreviously)
		}
	}

	return nil
}
func (scp *schemaParser) parseInterfaceType(gofile *ast.File, bschema *spec.Schema, tpe *ast.InterfaceType, seenPreviously map[string]struct{}) error {
	if tpe.Methods == nil {
		return nil
	}

	// first check if this has embedded interfaces, if so make sure to refer to those by ref
	// when they are decorated with an allOf annotation
	// go over the method list again and this time collect the nullary methods and parse the comments
	// as if they are properties on a struct
	var schema *spec.Schema
	seenProperties := seenPreviously
	hasAllOf := false

	for _, fld := range tpe.Methods.List {
		if len(fld.Names) == 0 {
			// if this created an allOf property then we have to rejig the schema var
			// because all the fields collected that aren't from embedded structs should go in
			// their own proper schema
			// first process embedded structs in order of embedding
			if allOfMember(fld.Doc) {
				hasAllOf = true
				if schema == nil {
					schema = new(spec.Schema)
				}
				var newSch spec.Schema
				// when the embedded struct is annotated with swagger:allOf it will be used as allOf property
				// otherwise the fields will just be included as normal properties
				if err := scp.parseAllOfMember(gofile, &newSch, fld.Type, seenProperties); err != nil {
					return err
				}

				if fld.Doc != nil {
					for _, cmt := range fld.Doc.List {
						for _, ln := range strings.Split(cmt.Text, "\n") {
							matches := rxAllOf.FindStringSubmatch(ln)
							ml := len(matches)
							if ml > 1 {
								mv := matches[ml-1]
								if mv != "" {
									bschema.AddExtension("x-class", mv)
								}
							}
						}
					}
				}

				bschema.AllOf = append(bschema.AllOf, newSch)
				continue
			}

			var newSch spec.Schema
			// when the embedded struct is annotated with swagger:allOf it will be used as allOf property
			// otherwise the fields will just be included as normal properties
			if err := scp.parseEmbeddedType(gofile, &newSch, fld.Type, seenProperties); err != nil {
				return err
			}
			bschema.AllOf = append(bschema.AllOf, newSch)
			hasAllOf = true
		}
	}
	if schema == nil {
		schema = bschema
	}
	// then add and possibly override values
	if schema.Properties == nil {
		schema.Properties = make(map[string]spec.Schema)
	}
	schema.Typed("object", "")
	for _, fld := range tpe.Methods.List {
		if mtpe, ok := fld.Type.(*ast.FuncType); ok && mtpe.Params.NumFields() == 0 && mtpe.Results.NumFields() == 1 {
			gnm := fld.Names[0].Name
			nm := gnm
			if fld.Doc != nil {
				for _, cmt := range fld.Doc.List {
					for _, ln := range strings.Split(cmt.Text, "\n") {
						matches := rxName.FindStringSubmatch(ln)
						ml := len(matches)
						if ml > 1 {
							nm = matches[ml-1]
						}
					}
				}
			}

			ps := schema.Properties[nm]
			if err := parseProperty(scp, gofile, mtpe.Results.List[0].Type, schemaTypable{&ps, 0}); err != nil {
				return err
			}

			if err := scp.createParser(nm, schema, &ps, fld).Parse(fld.Doc); err != nil {
				return err
			}

			if nm != gnm {
				ps.AddExtension("x-go-name", gnm)
			}
			seenProperties[nm] = struct{}{}
			schema.Properties[nm] = ps
		}

	}
	if schema != nil && hasAllOf {
		bschema.AllOf = append(bschema.AllOf, *schema)
	}
	for k := range schema.Properties {
		if _, ok := seenProperties[k]; !ok {
			delete(schema.Properties, k)
		}
	}
	return nil
}

func (scp *schemaParser) parseStructType(gofile *ast.File, bschema *spec.Schema, tpe *ast.StructType, seenPreviously map[string]struct{}) error {
	if tpe.Fields == nil {
		return nil
	}
	var schema *spec.Schema
	seenProperties := seenPreviously
	hasAllOf := false

	for _, fld := range tpe.Fields.List {
		if len(fld.Names) == 0 {
			// if this created an allOf property then we have to rejig the schema var
			// because all the fields collected that aren't from embedded structs should go in
			// their own proper schema
			// first process embedded structs in order of embedding
			if allOfMember(fld.Doc) {
				hasAllOf = true
				if schema == nil {
					schema = new(spec.Schema)
				}
				var newSch spec.Schema
				// when the embedded struct is annotated with swagger:allOf it will be used as allOf property
				// otherwise the fields will just be included as normal properties
				if err := scp.parseAllOfMember(gofile, &newSch, fld.Type, seenProperties); err != nil {
					return err
				}

				if fld.Doc != nil {
					for _, cmt := range fld.Doc.List {
						for _, ln := range strings.Split(cmt.Text, "\n") {
							matches := rxAllOf.FindStringSubmatch(ln)
							ml := len(matches)
							if ml > 1 {
								mv := matches[ml-1]
								if mv != "" {
									bschema.AddExtension("x-class", mv)
								}
							}
						}
					}
				}

				bschema.AllOf = append(bschema.AllOf, newSch)
				continue
			}
			if schema == nil {
				schema = bschema
			}

			// when the embedded struct is annotated with swagger:allOf it will be used as allOf property
			// otherwise the fields will just be included as normal properties
			if err := scp.parseEmbeddedType(gofile, schema, fld.Type, seenProperties); err != nil {
				return err
			}
		}
	}
	if schema == nil {
		schema = bschema
	}

	// then add and possibly override values
	if schema.Properties == nil {
		schema.Properties = make(map[string]spec.Schema)
	}
	schema.Typed("object", "")
	for _, fld := range tpe.Fields.List {
		var tag string
		if fld.Tag != nil {
			val, err := strconv.Unquote(fld.Tag.Value)
			if err == nil {
				tag = reflect.StructTag(val).Get("json")
			}
		}
		if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() && (tag == "" || tag[0] != '-') {
			var nm, gnm string
			nm = fld.Names[0].Name
			gnm = nm
			if fld.Tag != nil && len(strings.TrimSpace(fld.Tag.Value)) > 0 /*&& fld.Tag.Value[0] != '-'*/ {
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

			ps := schema.Properties[nm]
			if err := parseProperty(scp, gofile, fld.Type, schemaTypable{&ps, 0}); err != nil {
				return err
			}

			if err := scp.createParser(nm, schema, &ps, fld).Parse(fld.Doc); err != nil {
				return err
			}

			if nm != gnm {
				ps.AddExtension("x-go-name", gnm)
			}
			seenProperties[nm] = struct{}{}
			schema.Properties[nm] = ps
		}
	}
	if schema != nil && hasAllOf {
		bschema.AllOf = append(bschema.AllOf, *schema)
	}
	for k := range schema.Properties {
		if _, ok := seenProperties[k]; !ok {
			delete(schema.Properties, k)
		}
	}
	return nil
}

func (scp *schemaParser) createParser(nm string, schema, ps *spec.Schema, fld *ast.Field) *sectionedParser {

	sp := new(sectionedParser)
	sp.setDescription = func(lines []string) { ps.Description = joinDropLast(lines) }
	if ps.Ref.String() == "" {
		sp.taggers = []tagParser{
			newSingleLineTagParser("maximum", &setMaximum{schemaValidations{ps}, rxf(rxMaximumFmt, "")}),
			newSingleLineTagParser("minimum", &setMinimum{schemaValidations{ps}, rxf(rxMinimumFmt, "")}),
			newSingleLineTagParser("multipleOf", &setMultipleOf{schemaValidations{ps}, rxf(rxMultipleOfFmt, "")}),
			newSingleLineTagParser("minLength", &setMinLength{schemaValidations{ps}, rxf(rxMinLengthFmt, "")}),
			newSingleLineTagParser("maxLength", &setMaxLength{schemaValidations{ps}, rxf(rxMaxLengthFmt, "")}),
			newSingleLineTagParser("pattern", &setPattern{schemaValidations{ps}, rxf(rxPatternFmt, "")}),
			newSingleLineTagParser("minItems", &setMinItems{schemaValidations{ps}, rxf(rxMinItemsFmt, "")}),
			newSingleLineTagParser("maxItems", &setMaxItems{schemaValidations{ps}, rxf(rxMaxItemsFmt, "")}),
			newSingleLineTagParser("unique", &setUnique{schemaValidations{ps}, rxf(rxUniqueFmt, "")}),
			newSingleLineTagParser("required", &setRequiredSchema{schema, nm}),
			newSingleLineTagParser("readOnly", &setReadOnlySchema{ps}),
			newSingleLineTagParser("discriminator", &setDiscriminator{schema, nm}),
		}

		itemsTaggers := func(items *spec.Schema, level int) []tagParser {
			// the expression is 1-index based not 0-index
			itemsPrefix := fmt.Sprintf(rxItemsPrefixFmt, level+1)
			return []tagParser{
				newSingleLineTagParser(fmt.Sprintf("items%dMaximum", level), &setMaximum{schemaValidations{items}, rxf(rxMaximumFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMinimum", level), &setMinimum{schemaValidations{items}, rxf(rxMinimumFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMultipleOf", level), &setMultipleOf{schemaValidations{items}, rxf(rxMultipleOfFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMinLength", level), &setMinLength{schemaValidations{items}, rxf(rxMinLengthFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMaxLength", level), &setMaxLength{schemaValidations{items}, rxf(rxMaxLengthFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dPattern", level), &setPattern{schemaValidations{items}, rxf(rxPatternFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMinItems", level), &setMinItems{schemaValidations{items}, rxf(rxMinItemsFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dMaxItems", level), &setMaxItems{schemaValidations{items}, rxf(rxMaxItemsFmt, itemsPrefix)}),
				newSingleLineTagParser(fmt.Sprintf("items%dUnique", level), &setUnique{schemaValidations{items}, rxf(rxUniqueFmt, itemsPrefix)}),
			}

		}
		// check if this is a primitive, if so parse the validations from the
		// doc comments of the slice declaration.
		if ftped, ok := fld.Type.(*ast.ArrayType); ok {
			ftpe := ftped
			items, level := ps.Items, 0
			for items != nil && items.Schema != nil {
				switch iftpe := ftpe.Elt.(type) {
				case *ast.ArrayType:
					eleTaggers := itemsTaggers(items.Schema, level)
					sp.taggers = append(eleTaggers, sp.taggers...)
					ftpe = iftpe
				case *ast.Ident:
					if iftpe.Obj == nil {
						sp.taggers = append(itemsTaggers(items.Schema, level), sp.taggers...)
					}
					break
					//default:
					//return fmt.Errorf("unknown field type (%T) ele for %q", iftpe, nm)
				}
				items = items.Schema.Items
				level = level + 1
			}
		}
	} else {
		sp.taggers = []tagParser{
			newSingleLineTagParser("required", &setRequiredSchema{schema, nm}),
		}
	}
	return sp
}

func (scp *schemaParser) packageForFile(gofile *ast.File) (*loader.PackageInfo, error) {
	for pkg, pkgInfo := range scp.program.AllPackages {
		if pkg.Name() == gofile.Name.Name {
			return pkgInfo, nil
		}
	}
	fn := scp.program.Fset.File(gofile.Pos()).Name()
	return nil, fmt.Errorf("unable to determine package for %s", fn)
}

func (scp *schemaParser) packageForSelector(gofile *ast.File, expr ast.Expr) (*loader.PackageInfo, error) {

	if pth, ok := expr.(*ast.Ident); ok {
		// lookup import
		var selPath string
		for _, imp := range gofile.Imports {
			pv, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				pv = imp.Path.Value
			}
			if imp.Name != nil {
				if imp.Name.Name == pth.Name {
					selPath = pv
					break
				}
			} else {
				parts := strings.Split(pv, "/")
				if len(parts) > 0 && parts[len(parts)-1] == pth.Name {
					selPath = pv
					break
				}
			}
		}
		// find actual struct
		if selPath == "" {
			return nil, fmt.Errorf("no import found for %s", pth.Name)
		}

		pkg := scp.program.Package(selPath)
		if pkg == nil {
			// TODO: I must admit this made me cry, it's not even a great solution.
			pkg = scp.program.Package("github.com/go-swagger/go-swagger/vendor/" + selPath)
			if pkg == nil {
				return nil, fmt.Errorf("no package found for %s", selPath)
			}
		}
		return pkg, nil
	}
	return nil, fmt.Errorf("can't determine selector path from %v", expr)
}

func (scp *schemaParser) parseIdentProperty(pkg *loader.PackageInfo, expr *ast.Ident, prop swaggerTypable) error {
	// find the file this selector points to
	file, gd, ts, err := findSourceFile(pkg, expr.Name)
	if err != nil {
		return swaggerSchemaForType(expr.Name, prop)
	}
	if at, ok := ts.Type.(*ast.ArrayType); ok {
		// the swagger spec defines strfmt base64 as []byte.
		// in that case we don't actually want to turn it into an array
		// but we want to turn it into a string
		if _, ok := at.Elt.(*ast.Ident); ok {
			if strfmtName, ok := strfmtName(gd.Doc); ok {
				prop.Typed("string", strfmtName)
				return nil
			}
		}
		// this is a selector, so most likely not base64
		if strfmtName, ok := strfmtName(gd.Doc); ok {
			prop.Items().Typed("string", strfmtName)
			return nil
		}
	}

	// look at doc comments for swagger:strfmt [name]
	// when found this is the format name, create a schema with that name
	if strfmtName, ok := strfmtName(gd.Doc); ok {
		prop.Typed("string", strfmtName)
		return nil
	}
	switch tpe := ts.Type.(type) {
	case *ast.ArrayType:
		switch atpe := tpe.Elt.(type) {
		case *ast.Ident:
			return scp.parseIdentProperty(pkg, atpe, prop.Items())
		case *ast.SelectorExpr:
			return scp.typeForSelector(file, atpe, prop.Items())
		default:
			return fmt.Errorf("unknown selector type: %#v", tpe)
		}
	case *ast.StructType:
		sd := newSchemaDecl(file, gd, ts)
		sd.inferNames()
		ref, err := spec.NewRef("#/definitions/" + sd.Name)
		if err != nil {
			return err
		}
		prop.SetRef(ref)
		scp.postDecls = append(scp.postDecls, *sd)
		return nil

	case *ast.Ident:
		return scp.parseIdentProperty(pkg, tpe, prop)

	case *ast.SelectorExpr:
		return scp.typeForSelector(file, tpe, prop)

	default:
		return swaggerSchemaForType(expr.Name, prop)
	}

}

func (scp *schemaParser) typeForSelector(gofile *ast.File, expr *ast.SelectorExpr, prop swaggerTypable) error {
	pkg, err := scp.packageForSelector(gofile, expr.X)
	if err != nil {
		return err
	}

	return scp.parseIdentProperty(pkg, expr.Sel, prop)
}

func findSourceFile(pkg *loader.PackageInfo, typeName string) (*ast.File, *ast.GenDecl, *ast.TypeSpec, error) {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok {
				for _, gs := range gd.Specs {
					if ts, ok := gs.(*ast.TypeSpec); ok {
						strfmtNme, isStrfmt := strfmtName(gd.Doc)
						if (isStrfmt && strfmtNme == typeName) || ts.Name != nil && ts.Name.Name == typeName {
							return file, gd, ts, nil
						}
					}
				}
			}
		}
	}
	return nil, nil, nil, fmt.Errorf("unable to find %s in %s", typeName, pkg.String())
}

func allOfMember(comments *ast.CommentGroup) bool {
	if comments != nil {
		for _, cmt := range comments.List {
			for _, ln := range strings.Split(cmt.Text, "\n") {
				if rxAllOf.MatchString(ln) {
					return true
				}
			}
		}
	}
	return false
}

func fileParam(comments *ast.CommentGroup) bool {
	if comments != nil {
		for _, cmt := range comments.List {
			for _, ln := range strings.Split(cmt.Text, "\n") {
				if rxFileUpload.MatchString(ln) {
					return true
				}
			}
		}
	}
	return false
}

func strfmtName(comments *ast.CommentGroup) (string, bool) {
	if comments != nil {
		for _, cmt := range comments.List {
			for _, ln := range strings.Split(cmt.Text, "\n") {
				matches := rxStrFmt.FindStringSubmatch(ln)
				if len(matches) > 1 && len(strings.TrimSpace(matches[1])) > 0 {
					return strings.TrimSpace(matches[1]), true
				}
			}
		}
	}
	return "", false
}

func parseProperty(scp *schemaParser, gofile *ast.File, fld ast.Expr, prop swaggerTypable) error {
	switch ftpe := fld.(type) {
	case *ast.Ident: // simple value
		pkg, err := scp.packageForFile(gofile)
		if err != nil {
			return err
		}
		return scp.parseIdentProperty(pkg, ftpe, prop)

	case *ast.StarExpr: // pointer to something, optional by default
		parseProperty(scp, gofile, ftpe.X, prop)

	case *ast.ArrayType: // slice type
		if err := parseProperty(scp, gofile, ftpe.Elt, prop.Items()); err != nil {
			return err
		}

	case *ast.StructType:
		schema := prop.Schema()
		if schema == nil {
			return fmt.Errorf("items doesn't support embedded structs")
		}
		return scp.parseStructType(gofile, prop.Schema(), ftpe, make(map[string]struct{}))

	case *ast.SelectorExpr:
		err := scp.typeForSelector(gofile, ftpe, prop)
		return err

	case *ast.MapType:
		// check if key is a string type, if not print a message
		// and skip the map property. Only maps with string keys can go into additional properties
		sch := prop.Schema()
		if sch == nil {
			return fmt.Errorf("items doesn't support maps")
		}
		if keyIdent, ok := ftpe.Key.(*ast.Ident); sch != nil && ok {
			if keyIdent.Name == "string" {
				if sch.AdditionalProperties == nil {
					sch.AdditionalProperties = new(spec.SchemaOrBool)
				}
				sch.AdditionalProperties.Allows = false
				if sch.AdditionalProperties.Schema == nil {
					sch.AdditionalProperties.Schema = new(spec.Schema)
				}
				parseProperty(scp, gofile, ftpe.Value, schemaTypable{sch.AdditionalProperties.Schema, 0})
				sch.Typed("object", "")
			}
		}

	case *ast.InterfaceType:
		prop.Schema().Typed("object", "")
	default:
		return fmt.Errorf("%s is unsupported for a schema", ftpe)
	}
	return nil
}
