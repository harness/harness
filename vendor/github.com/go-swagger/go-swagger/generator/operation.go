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

package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-swagger/go-swagger/httpkit"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/swag"
)

// GenerateServerOperation generates a parameter model, parameter validator, http handler implementations for a given operation
// It also generates an operation handler interface that uses the parameter model for handling a valid request.
// Allows for specifying a list of tags to include only certain tags for the generation
func GenerateServerOperation(operationNames, tags []string, includeHandler, includeParameters, includeResponses bool, opts GenOpts) error {

	if opts.TemplateDir != "" {
		if err := templates.LoadDir(opts.TemplateDir); err != nil {
			return err
		}
	}

	compileTemplates()

	// Load the spec
	_, specDoc, err := loadSpec(opts.Spec)
	if err != nil {
		return err
	}

	ops := gatherOperations(specDoc, operationNames)

	for operationName, opRef := range ops {
		method, path, operation := opRef.Method, opRef.Path, opRef.Op
		defaultScheme := opts.DefaultScheme
		if defaultScheme == "" {
			defaultScheme = "http"
		}
		defaultProduces := opts.DefaultProduces
		if defaultProduces == "" {
			defaultProduces = "application/json"
		}
		defaultConsumes := opts.DefaultConsumes
		if defaultConsumes == "" {
			defaultConsumes = "application/json"
		}

		apiPackage := mangleName(swag.ToFileName(opts.APIPackage), "api")
		serverPackage := mangleName(swag.ToFileName(opts.ServerPackage), "server")
		generator := operationGenerator{
			Name:                 operationName,
			Method:               method,
			Path:                 path,
			APIPackage:           apiPackage,
			ModelsPackage:        mangleName(swag.ToFileName(opts.ModelPackage), "definitions"),
			ClientPackage:        mangleName(swag.ToFileName(opts.ClientPackage), "client"),
			ServerPackage:        serverPackage,
			Operation:            *operation,
			SecurityRequirements: specDoc.SecurityRequirementsFor(operation),
			Principal:            opts.Principal,
			Target:               filepath.Join(opts.Target, serverPackage),
			Base:                 opts.Target,
			Tags:                 tags,
			IncludeHandler:       includeHandler,
			IncludeParameters:    includeParameters,
			IncludeResponses:     includeResponses,
			DumpData:             opts.DumpData,
			DefaultScheme:        defaultScheme,
			DefaultProduces:      defaultProduces,
			Doc:                  specDoc,
		}
		if err := generator.Generate(); err != nil {
			return err
		}
	}
	return nil
}

type operationGenerator struct {
	Name                 string
	Method               string
	Path                 string
	Authorized           bool
	APIPackage           string
	ModelsPackage        string
	ServerPackage        string
	ClientPackage        string
	Operation            spec.Operation
	SecurityRequirements []spec.SecurityRequirement
	Principal            string
	Target               string
	Base                 string
	Tags                 []string
	data                 interface{}
	pkg                  string
	cname                string
	IncludeHandler       bool
	IncludeParameters    bool
	IncludeResponses     bool
	DumpData             bool
	DefaultScheme        string
	DefaultProduces      string
	Doc                  *spec.Document
	WithContext          bool
}

func (o *operationGenerator) Generate() error {
	// Build a list of codegen operations based on the tags,
	// the tag decides the actual package for an operation
	// the user specified package serves as root for generating the directory structure
	var operations GenOperations
	authed := len(o.SecurityRequirements) > 0

	var bldr codeGenOpBuilder
	bldr.Name = o.Name
	bldr.Method = o.Method
	bldr.Path = o.Path
	bldr.ModelsPackage = o.ModelsPackage
	bldr.Principal = o.Principal
	bldr.Target = o.Target
	bldr.Operation = o.Operation
	bldr.Authed = authed
	bldr.Doc = o.Doc
	bldr.DefaultScheme = o.DefaultScheme
	bldr.DefaultProduces = o.DefaultProduces
	bldr.DefaultImports = []string{filepath.ToSlash(filepath.Join(baseImport(o.Base), o.ModelsPackage))}
	bldr.RootAPIPackage = o.APIPackage
	bldr.WithContext = o.WithContext

	for _, tag := range o.Operation.Tags {
		if len(o.Tags) == 0 {
			bldr.APIPackage = mangleName(swag.ToFileName(tag), o.APIPackage)
			op, err := bldr.MakeOperation()
			if err != nil {
				return err
			}
			operations = append(operations, op)
			continue
		}
		for _, ft := range o.Tags {
			if ft == tag {
				bldr.APIPackage = mangleName(swag.ToFileName(tag), o.APIPackage)
				op, err := bldr.MakeOperation()
				if err != nil {
					return err
				}
				operations = append(operations, op)
				break
			}
		}
	}
	if len(operations) == 0 {
		bldr.APIPackage = o.APIPackage
		op, err := bldr.MakeOperation()
		if err != nil {
			return err
		}
		operations = append(operations, op)
	}
	sort.Sort(operations)

	for _, op := range operations {
		if o.DumpData {
			bb, _ := json.MarshalIndent(swag.ToDynamicJSON(op), "", " ")
			fmt.Fprintln(os.Stdout, string(bb))
			continue
		}
		og := new(opGen)
		og.IncludeHandler = o.IncludeHandler
		og.IncludeParameters = o.IncludeParameters
		og.IncludeResponses = o.IncludeResponses
		og.data = &op
		og.pkg = op.Package
		og.cname = swag.ToGoName(op.Name)
		og.Doc = o.Doc
		og.Target = o.Target
		og.APIPackage = o.APIPackage
		og.WithContext = o.WithContext
		return og.Generate()
	}

	return nil
}

type opGen struct {
	data              *GenOperation
	pkg               string
	cname             string
	IncludeHandler    bool
	IncludeParameters bool
	IncludeResponses  bool
	Doc               *spec.Document
	Target            string
	APIPackage        string
	WithContext       bool
}

func (o *opGen) Generate() error {

	if o.IncludeHandler {
		if err := o.generateHandler(); err != nil {
			return fmt.Errorf("handler: %s", err)
		}
		log.Println("generated handler", o.data.Package+"."+o.cname)
	}

	opParams := o.Doc.ParamsFor(o.data.Method, o.data.Path)
	if o.IncludeParameters && len(opParams) > 0 {
		if err := o.generateParameterModel(); err != nil {
			return fmt.Errorf("parameters: %s", err)
		}
		log.Println("generated parameters", o.data.Package+"."+o.cname+"Parameters")
	}

	if o.IncludeResponses && len(o.data.Responses) > 0 {
		if err := o.generateResponses(); err != nil {
			return fmt.Errorf("responses: %s", err)
		}
		log.Println("generated responses", o.data.Package+"."+o.cname+"Responses")
	}

	if len(opParams) == 0 {
		log.Println("no parameters for operation", o.data.Package+"."+o.cname)
	}
	return nil
}

func (o *opGen) generateHandler() error {
	buf := bytes.NewBuffer(nil)

	if err := operationTemplate.Execute(buf, o.data); err != nil {
		return err
	}
	log.Println("rendered handler template:", o.pkg+"."+o.cname)

	fp := filepath.Join(o.Target, o.pkg)
	if o.pkg != o.APIPackage {
		fp = filepath.Join(o.Target, o.APIPackage, o.pkg)
	}
	return writeToFile(fp, swag.ToGoName(o.data.Name), buf.Bytes())
}

func (o *opGen) generateParameterModel() error {
	buf := bytes.NewBuffer(nil)

	if err := parameterTemplate.Execute(buf, o.data); err != nil {
		return err
	}
	log.Println("rendered parameters template:", o.pkg+"."+o.cname+"Parameters")

	fp := filepath.Join(o.Target, o.pkg)
	if o.pkg != o.APIPackage {
		fp = filepath.Join(o.Target, o.APIPackage, o.pkg)
	}
	return writeToFile(fp, swag.ToGoName(o.data.Name)+"Parameters", buf.Bytes())
}

func (o *opGen) generateResponses() error {
	buf := bytes.NewBuffer(nil)

	if err := responsesTemplate.Execute(buf, o.data); err != nil {
		return err
	}
	log.Println("rendered responses template:", o.pkg+"."+o.cname+"Responses")

	fp := filepath.Join(o.Target, o.pkg)
	if o.pkg != o.APIPackage {
		fp = filepath.Join(o.Target, o.APIPackage, o.pkg)
	}
	return writeToFile(fp, swag.ToGoName(o.data.Name)+"Responses", buf.Bytes())
}

type codeGenOpBuilder struct {
	Name            string
	Method          string
	Path            string
	APIPackage      string
	RootAPIPackage  string
	ModelsPackage   string
	Principal       string
	Target          string
	WithContext     bool
	Operation       spec.Operation
	Doc             *spec.Document
	Authed          bool
	DefaultImports  []string
	DefaultScheme   string
	DefaultProduces string
	DefaultConsumes string
	ExtraSchemas    map[string]GenSchema
	origDefs        map[string]spec.Schema
}

func (b *codeGenOpBuilder) MakeOperation() (GenOperation, error) {
	if Debug {
		log.Printf("[%s %s] parsing operation (id: %q)", b.Method, b.Path, b.Operation.ID)
	}
	resolver := newTypeResolver(b.ModelsPackage, b.Doc.ResetDefinitions())
	receiver := "o"

	operation := b.Operation
	var params, qp, pp, hp, fp GenParameters
	var hasQueryParams, hasFormParams, hasFileParams bool
	for _, p := range b.Doc.ParamsFor(b.Method, b.Path) {
		cp, err := b.MakeParameter(receiver, resolver, p)
		if err != nil {
			return GenOperation{}, err
		}
		if cp.IsQueryParam() {
			hasQueryParams = true
			qp = append(qp, cp)
		}
		if cp.IsFormParam() {
			if p.Type == "file" {
				hasFileParams = true
			}
			hasFormParams = true
			fp = append(fp, cp)
		}
		if cp.IsPathParam() {
			pp = append(pp, cp)
		}
		if cp.IsHeaderParam() {
			hp = append(hp, cp)
		}
		params = append(params, cp)
	}
	sort.Sort(params)
	sort.Sort(qp)
	sort.Sort(pp)
	sort.Sort(hp)
	sort.Sort(fp)

	var responses map[int]GenResponse
	var defaultResponse *GenResponse
	var successResponse *GenResponse
	if operation.Responses != nil {
		for k, v := range operation.Responses.StatusCodeResponses {
			isSuccess := k/100 == 2
			gr, err := b.MakeResponse(receiver, swag.ToJSONName(b.Name+" "+httpkit.Statuses[k]), isSuccess, resolver, k, v)
			if err != nil {
				return GenOperation{}, err
			}
			if isSuccess {
				successResponse = &gr
			}
			if responses == nil {
				responses = make(map[int]GenResponse)
			}
			responses[k] = gr
		}

		if operation.Responses.Default != nil {
			gr, err := b.MakeResponse(receiver, b.Name+" default", false, resolver, -1, *operation.Responses.Default)
			if err != nil {
				return GenOperation{}, err
			}
			defaultResponse = &gr
		}
	}

	prin := b.Principal
	if prin == "" {
		prin = "interface{}"
	}

	var extra []GenSchema
	for _, sch := range b.ExtraSchemas {
		extra = append(extra, sch)
	}

	swsp := resolver.Doc.Spec()
	var extraSchemes []string
	if ess, ok := operation.Extensions.GetStringSlice("x-schemes"); ok {
		extraSchemes = append(extraSchemes, ess...)
	}

	if ess1, ok := swsp.Extensions.GetStringSlice("x-schemes"); ok {
		extraSchemes = concatUnique(ess1, extraSchemes)
	}
	sort.Strings(extraSchemes)
	schemes := concatUnique(swsp.Schemes, operation.Schemes)
	sort.Strings(schemes)
	produces := producesOrDefault(operation.Produces, swsp.Produces, b.DefaultProduces)
	sort.Strings(produces)
	consumes := producesOrDefault(operation.Consumes, swsp.Consumes, b.DefaultConsumes)
	sort.Strings(consumes)

	return GenOperation{
		Package:            b.APIPackage,
		RootPackage:        b.RootAPIPackage,
		Name:               b.Name,
		Method:             b.Method,
		Path:               b.Path,
		Tags:               operation.Tags[:],
		Description:        operation.Description,
		ReceiverName:       receiver,
		DefaultImports:     b.DefaultImports,
		Params:             params,
		Summary:            operation.Summary,
		QueryParams:        qp,
		PathParams:         pp,
		HeaderParams:       hp,
		FormParams:         fp,
		HasQueryParams:     hasQueryParams,
		HasFormParams:      hasFormParams,
		HasFileParams:      hasFileParams,
		Authorized:         b.Authed,
		Principal:          prin,
		Responses:          responses,
		DefaultResponse:    defaultResponse,
		SuccessResponse:    successResponse,
		ExtraSchemas:       extra,
		Schemes:            schemeOrDefault(schemes, b.DefaultScheme),
		ProducesMediaTypes: produces,
		ConsumesMediaTypes: consumes,
		ExtraSchemes:       extraSchemes,
		WithContext:        b.WithContext,
	}, nil
}

func producesOrDefault(produces []string, fallback []string, defaultProduces string) []string {
	if len(produces) > 0 {
		return produces
	}
	if len(fallback) > 0 {
		return fallback
	}
	return []string{defaultProduces}
}

func schemeOrDefault(schemes []string, defaultScheme string) []string {
	if len(schemes) == 0 {
		return []string{defaultScheme}
	}
	return schemes
}

func concatUnique(collections ...[]string) []string {
	resultSet := make(map[string]struct{})
	for _, c := range collections {
		for _, i := range c {
			if _, ok := resultSet[i]; !ok {
				resultSet[i] = struct{}{}
			}
		}
	}
	var result []string
	for k := range resultSet {
		result = append(result, k)
	}
	return result
}

func (b *codeGenOpBuilder) MakeResponse(receiver, name string, isSuccess bool, resolver *typeResolver, code int, resp spec.Response) (GenResponse, error) {
	if Debug {
		log.Printf("[%s %s] making id %q", b.Method, b.Path, b.Operation.ID)
	}
	res := GenResponse{
		Package:        b.APIPackage,
		ModelsPackage:  b.ModelsPackage,
		ReceiverName:   receiver,
		Name:           name,
		Description:    resp.Description,
		DefaultImports: nil,
		Imports:        nil,
		IsSuccess:      isSuccess,
		Code:           code,
		Method:         b.Method,
		Path:           b.Path,
	}

	for hName, header := range resp.Headers {
		res.Headers = append(res.Headers, b.MakeHeader(receiver, hName, header))
	}
	sort.Sort(res.Headers)

	if resp.Schema != nil {
		sc := schemaGenContext{
			Path:         fmt.Sprintf("%q", name),
			Name:         name + "Body",
			Receiver:     receiver,
			ValueExpr:    receiver,
			IndexVar:     "i",
			Schema:       *resp.Schema,
			Required:     true,
			TypeResolver: resolver,
			Named:        false,
			ExtraSchemas: make(map[string]GenSchema),
		}
		if err := sc.makeGenSchema(); err != nil {
			return GenResponse{}, err
		}

		for k, v := range sc.ExtraSchemas {
			if b.ExtraSchemas == nil {
				b.ExtraSchemas = make(map[string]GenSchema)
			}
			b.ExtraSchemas[k] = v
		}

		schema := sc.GenSchema
		if schema.IsAnonymous {

			schema.Name = swag.ToGoName(sc.Name + " Body")
			nm := schema.Name
			if b.ExtraSchemas == nil {
				b.ExtraSchemas = make(map[string]GenSchema)
			}
			b.ExtraSchemas[schema.Name] = schema
			schema = GenSchema{}
			schema.IsAnonymous = false
			schema.GoType = resolver.goTypeName(nm)
			schema.SwaggerType = nm
		}

		res.Schema = &schema
	}
	return res, nil
}

func (b *codeGenOpBuilder) MakeHeader(receiver, name string, hdr spec.Header) GenHeader {
	hasNumberValidation := hdr.Maximum != nil || hdr.Minimum != nil || hdr.MultipleOf != nil
	hasStringValidation := hdr.MaxLength != nil || hdr.MinLength != nil || hdr.Pattern != ""
	hasSliceValidations := hdr.MaxItems != nil || hdr.MinItems != nil || hdr.UniqueItems
	hasValidations := hasNumberValidation || hasStringValidation || hasSliceValidations || len(hdr.Enum) > 0

	tpe := typeForHeader(hdr) //simpleResolvedType(hdr.Type, hdr.Format, hdr.Items)

	return GenHeader{
		sharedValidations: sharedValidations{
			Required:            true,
			Maximum:             hdr.Maximum,
			ExclusiveMaximum:    hdr.ExclusiveMaximum,
			Minimum:             hdr.Minimum,
			ExclusiveMinimum:    hdr.ExclusiveMinimum,
			MaxLength:           hdr.MaxLength,
			MinLength:           hdr.MinLength,
			Pattern:             hdr.Pattern,
			MaxItems:            hdr.MaxItems,
			MinItems:            hdr.MinItems,
			UniqueItems:         hdr.UniqueItems,
			MultipleOf:          hdr.MultipleOf,
			Enum:                hdr.Enum,
			HasValidations:      hasValidations,
			HasSliceValidations: hasSliceValidations,
		},
		resolvedType: tpe,
		Package:      b.APIPackage,
		ReceiverName: receiver,
		Name:         name,
		Path:         fmt.Sprintf("%q", name),
		Description:  hdr.Description,
		Default:      hdr.Default,
		Converter:    stringConverters[tpe.GoType],
		Formatter:    stringFormatters[tpe.GoType],
	}
}

func (b *codeGenOpBuilder) MakeParameterItem(receiver, paramName, indexVar, path, valueExpression, location string, resolver *typeResolver, items, parent *spec.Items) (GenItems, error) {
	var res GenItems
	res.resolvedType = simpleResolvedType(items.Type, items.Format, items.Items)
	res.sharedValidations = sharedValidations{
		Maximum:          items.Maximum,
		ExclusiveMaximum: items.ExclusiveMaximum,
		Minimum:          items.Minimum,
		ExclusiveMinimum: items.ExclusiveMinimum,
		MaxLength:        items.MaxLength,
		MinLength:        items.MinLength,
		Pattern:          items.Pattern,
		MaxItems:         items.MaxItems,
		MinItems:         items.MinItems,
		UniqueItems:      items.UniqueItems,
		MultipleOf:       items.MultipleOf,
		Enum:             items.Enum,
	}
	res.Name = paramName
	res.Path = path
	res.Location = location
	res.ValueExpression = valueExpression
	res.CollectionFormat = items.CollectionFormat
	res.Converter = stringConverters[res.GoType]
	res.Formatter = stringFormatters[res.GoType]

	if items.Items != nil {
		pi, err := b.MakeParameterItem(receiver, paramName+" "+indexVar, indexVar+"[i]", "fmt.Sprintf(\"%s.%v\", "+path+", "+indexVar+")", valueExpression+"["+indexVar+"]", location, resolver, items.Items, items)
		if err != nil {
			return GenItems{}, err
		}
		res.Child = &pi
		pi.Parent = &res
	}

	return res, nil
}

func (b *codeGenOpBuilder) MakeParameter(receiver string, resolver *typeResolver, param spec.Parameter) (GenParameter, error) {
	if Debug {
		log.Printf("[%s %s] making parameter %q", b.Method, b.Path, param.Name)
	}
	var child *GenItems
	res := GenParameter{
		Name:             param.Name,
		ModelsPackage:    b.ModelsPackage,
		Path:             fmt.Sprintf("%q", param.Name),
		ValueExpression:  fmt.Sprintf("%s.%s", receiver, pascalize(param.Name)),
		IndexVar:         "i",
		BodyParam:        nil,
		Default:          param.Default,
		Enum:             param.Enum,
		Description:      param.Description,
		ReceiverName:     receiver,
		CollectionFormat: param.CollectionFormat,
		Child:            child,
		Location:         param.In,
		AllowEmptyValue:  (param.In == "query" || param.In == "formData") && param.AllowEmptyValue,
	}

	if param.In == "body" {
		sc := schemaGenContext{
			Path:         res.Path,
			Name:         res.Name,
			Receiver:     res.ReceiverName,
			ValueExpr:    res.ReceiverName,
			IndexVar:     res.IndexVar,
			Schema:       *param.Schema,
			Required:     param.Required,
			TypeResolver: resolver,
			Named:        false,
			ExtraSchemas: make(map[string]GenSchema),
		}
		if err := sc.makeGenSchema(); err != nil {
			return GenParameter{}, err
		}

		schema := sc.GenSchema
		if schema.IsAnonymous {
			schema.Name = swag.ToGoName(b.Operation.ID + " Body")
			nm := schema.Name
			schema.GoType = nm
			schema.IsAnonymous = false
			if len(schema.Properties) > 0 {
				if b.ExtraSchemas == nil {
					b.ExtraSchemas = make(map[string]GenSchema)
				}
				b.ExtraSchemas[nm] = schema
			}
			prevSchema := schema
			schema = GenSchema{}
			schema.IsAnonymous = false
			schema.GoType = nm
			schema.SwaggerType = nm
			if len(prevSchema.Properties) == 0 {
				schema.GoType = "interface{}"
			}
			schema.IsComplexObject = true
			schema.IsInterface = len(schema.Properties) == 0
		}
		res.Schema = &schema
		it := res.Schema.Items

		items := new(GenItems)
		var prev *GenItems
		next := items
		for it != nil {
			next.resolvedType = it.resolvedType
			next.sharedValidations = it.sharedValidations
			next.Formatter = stringFormatters[it.SwaggerFormat]
			_, next.IsCustomFormatter = customFormatters[it.SwaggerFormat]
			it = it.Items
			if prev != nil {
				prev.Child = next
			}
			prev = next
			next = new(GenItems)
		}
		res.Child = items
		res.resolvedType = schema.resolvedType
		res.sharedValidations = schema.sharedValidations
		res.ZeroValue = schema.Zero()

	} else {
		res.resolvedType = simpleResolvedType(param.Type, param.Format, param.Items)
		res.sharedValidations = sharedValidations{
			Required:         param.Required,
			Maximum:          param.Maximum,
			ExclusiveMaximum: param.ExclusiveMaximum,
			Minimum:          param.Minimum,
			ExclusiveMinimum: param.ExclusiveMinimum,
			MaxLength:        param.MaxLength,
			MinLength:        param.MinLength,
			Pattern:          param.Pattern,
			MaxItems:         param.MaxItems,
			MinItems:         param.MinItems,
			UniqueItems:      param.UniqueItems,
			MultipleOf:       param.MultipleOf,
			Enum:             param.Enum,
		}

		if param.Items != nil {
			pi, err := b.MakeParameterItem(receiver, param.Name+" "+res.IndexVar, res.IndexVar+"i", "fmt.Sprintf(\"%s.%v\", "+res.Path+", "+res.IndexVar+")", res.ValueExpression+"["+res.IndexVar+"]", param.In, resolver, param.Items, nil)
			if err != nil {
				return GenParameter{}, err
			}
			res.Child = &pi
		}
		res.IsNullable = !param.Required && !param.AllowEmptyValue

	}

	hasNumberValidation := param.Maximum != nil || param.Minimum != nil || param.MultipleOf != nil
	hasStringValidation := param.MaxLength != nil || param.MinLength != nil || param.Pattern != ""
	hasSliceValidations := param.MaxItems != nil || param.MinItems != nil || param.UniqueItems
	hasValidations := hasNumberValidation || hasStringValidation || hasSliceValidations || len(param.Enum) > 0

	res.Converter = stringConverters[res.GoType]
	res.Formatter = stringFormatters[res.GoType]
	res.HasValidations = hasValidations
	res.HasSliceValidations = hasSliceValidations
	return res, nil
}
