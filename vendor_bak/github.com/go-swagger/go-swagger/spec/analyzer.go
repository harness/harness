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

package spec

import (
	"fmt"
	slashpath "path"
	"strconv"
	"strings"

	"github.com/go-swagger/go-swagger/jsonpointer"
	"github.com/go-swagger/go-swagger/swag"
)

type referenceAnalysis struct {
	schemas    map[string]Ref
	responses  map[string]Ref
	parameters map[string]Ref
	referenced struct {
		schemas    map[string]SchemaRef
		responses  map[string]*Response
		parameters map[string]*Parameter
	}
}

func (r *referenceAnalysis) addSchemaRef(key string, ref SchemaRef) {
	if r.schemas == nil {
		r.schemas = make(map[string]Ref)
	}
	r.schemas["#"+key] = ref.Schema.Ref
}

func (r *referenceAnalysis) addResponseRef(key string, resp *Response) {
	if r.responses == nil {
		r.responses = make(map[string]Ref)
	}
	r.responses["#"+key] = resp.Ref
}

func (r *referenceAnalysis) addParamRef(key string, param *Parameter) {
	if r.parameters == nil {
		r.parameters = make(map[string]Ref)
	}
	r.parameters["#"+key] = param.Ref
}

// specAnalyzer takes a swagger spec object and turns it into a registry
// with a bunch of utility methods to act on the information in the spec
type specAnalyzer struct {
	spec        *Swagger
	consumes    map[string]struct{}
	produces    map[string]struct{}
	authSchemes map[string]struct{}
	operations  map[string]map[string]*Operation
	references  referenceAnalysis
	allSchemas  map[string]SchemaRef
	allOfs      map[string]SchemaRef
}

func (s *specAnalyzer) initialize() {
	for _, c := range s.spec.Consumes {
		s.consumes[c] = struct{}{}
	}
	for _, c := range s.spec.Produces {
		s.produces[c] = struct{}{}
	}
	for _, ss := range s.spec.Security {
		for k := range ss {
			s.authSchemes[k] = struct{}{}
		}
	}
	for path, pathItem := range s.AllPaths() {
		s.analyzeOperations(path, &pathItem)
	}

	for name, parameter := range s.spec.Parameters {
		if parameter.In == "body" && parameter.Schema != nil {
			s.analyzeSchema("schema", *parameter.Schema, slashpath.Join("/parameters", jsonpointer.Escape(name)))
		}
	}

	for name, response := range s.spec.Responses {
		if response.Schema != nil {
			s.analyzeSchema("schema", *response.Schema, slashpath.Join("/responses", jsonpointer.Escape(name)))
		}
	}

	for name, schema := range s.spec.Definitions {
		s.analyzeSchema(name, schema, "/definitions")
	}
	// TODO: after analyzing all things and flattening shemas etc
	// resolve all the collected references to their final representations
	// best put in a separate method because this could get expensive
}

func (s *specAnalyzer) analyzeOperations(path string, pi *PathItem) {
	// TODO: resolve refs here?
	op := pi
	s.analyzeOperation("GET", path, op.Get)
	s.analyzeOperation("PUT", path, op.Put)
	s.analyzeOperation("POST", path, op.Post)
	s.analyzeOperation("PATCH", path, op.Patch)
	s.analyzeOperation("DELETE", path, op.Delete)
	s.analyzeOperation("HEAD", path, op.Head)
	s.analyzeOperation("OPTIONS", path, op.Options)
	for i, param := range op.Parameters {
		refPref := slashpath.Join("/paths", jsonpointer.Escape(path), "parameters", strconv.Itoa(i))
		if param.Ref.String() != "" {
			s.references.addParamRef(refPref, &param)
		}
		if param.Schema != nil {
			s.analyzeSchema("schema", *param.Schema, refPref)
		}
	}
}

func (s *specAnalyzer) analyzeOperation(method, path string, op *Operation) {
	if op == nil {
		return
	}

	for _, c := range op.Consumes {
		s.consumes[c] = struct{}{}
	}
	for _, c := range op.Produces {
		s.produces[c] = struct{}{}
	}
	for _, ss := range op.Security {
		for k := range ss {
			s.authSchemes[k] = struct{}{}
		}
	}
	if _, ok := s.operations[method]; !ok {
		s.operations[method] = make(map[string]*Operation)
	}
	s.operations[method][path] = op
	prefix := slashpath.Join("/paths", jsonpointer.Escape(path), strings.ToLower(method))
	for i, param := range op.Parameters {
		refPref := slashpath.Join(prefix, "parameters", strconv.Itoa(i))
		if param.Ref.String() != "" {
			s.references.addParamRef(refPref, &param)
		}
		if param.In == "body" && param.Schema != nil {
			s.analyzeSchema("schema", *param.Schema, refPref)
		}
	}
	if op.Responses != nil {
		if op.Responses.Default != nil {
			refPref := slashpath.Join(prefix, "responses", "default")
			if op.Responses.Default.Ref.String() != "" {
				s.references.addResponseRef(refPref, op.Responses.Default)
			}
			if op.Responses.Default.Schema != nil {
				s.analyzeSchema("schema", *op.Responses.Default.Schema, refPref)
			}
		}
		for k, res := range op.Responses.StatusCodeResponses {
			refPref := slashpath.Join(prefix, "responses", strconv.Itoa(k))
			if res.Ref.String() != "" {
				s.references.addResponseRef(refPref, &res)
			}
			if res.Schema != nil {
				s.analyzeSchema("schema", *res.Schema, refPref)
			}
		}
	}
}

func (s *specAnalyzer) analyzeSchema(name string, schema Schema, prefix string) {
	refURI := slashpath.Join(prefix, jsonpointer.Escape(name))
	schRef := SchemaRef{
		Name:   name,
		Schema: &schema,
		Ref:    MustCreateRef("#" + refURI),
	}
	s.allSchemas["#"+refURI] = schRef
	if schema.Ref.String() != "" {
		s.references.addSchemaRef(refURI, schRef)
	}
	for k, v := range schema.Definitions {
		s.analyzeSchema(k, v, slashpath.Join(refURI, "definitions"))
	}
	for k, v := range schema.Properties {
		s.analyzeSchema(k, v, slashpath.Join(refURI, "properties"))
	}
	for k, v := range schema.PatternProperties {
		s.analyzeSchema(k, v, slashpath.Join(refURI, "patternProperties"))
	}
	for i, v := range schema.AllOf {
		s.analyzeSchema(strconv.Itoa(i), v, slashpath.Join(refURI, "allOf"))
	}
	if len(schema.AllOf) > 0 {
		s.allOfs["#"+refURI] = SchemaRef{Name: name, Schema: &schema, Ref: MustCreateRef("#" + refURI)}
	}
	for i, v := range schema.AnyOf {
		s.analyzeSchema(strconv.Itoa(i), v, slashpath.Join(refURI, "anyOf"))
	}
	for i, v := range schema.OneOf {
		s.analyzeSchema(strconv.Itoa(i), v, slashpath.Join(refURI, "oneOf"))
	}
	if schema.Not != nil {
		s.analyzeSchema("not", *schema.Not, refURI)
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		s.analyzeSchema("additionalProperties", *schema.AdditionalProperties.Schema, refURI)
	}
	if schema.AdditionalItems != nil && schema.AdditionalItems.Schema != nil {
		s.analyzeSchema("additionalItems", *schema.AdditionalItems.Schema, refURI)
	}
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			s.analyzeSchema("items", *schema.Items.Schema, refURI)
		}
		for i, sch := range schema.Items.Schemas {
			s.analyzeSchema(strconv.Itoa(i), sch, slashpath.Join(refURI, "items"))
		}
	}
}

// SecurityRequirement is a representation of a security requirement for an operation
type SecurityRequirement struct {
	Name   string
	Scopes []string
}

// SecurityRequirementsFor gets the security requirements for the operation
func (s *specAnalyzer) SecurityRequirementsFor(operation *Operation) []SecurityRequirement {
	if s.spec.Security == nil && operation.Security == nil {
		return nil
	}

	schemes := s.spec.Security
	if operation.Security != nil {
		schemes = operation.Security
	}

	unique := make(map[string]SecurityRequirement)
	for _, scheme := range schemes {
		for k, v := range scheme {
			if _, ok := unique[k]; !ok {
				unique[k] = SecurityRequirement{Name: k, Scopes: v}
			}
		}
	}

	var result []SecurityRequirement
	for _, v := range unique {
		result = append(result, v)
	}
	return result
}

// SecurityDefinitionsFor gets the matching security definitions for a set of requirements
func (s *specAnalyzer) SecurityDefinitionsFor(operation *Operation) map[string]SecurityScheme {
	requirements := s.SecurityRequirementsFor(operation)
	if len(requirements) == 0 {
		return nil
	}
	result := make(map[string]SecurityScheme)
	for _, v := range requirements {
		if definition, ok := s.spec.SecurityDefinitions[v.Name]; ok {
			if definition != nil {
				result[v.Name] = *definition
			}
		}
	}
	return result
}

// ConsumesFor gets the mediatypes for the operation
func (s *specAnalyzer) ConsumesFor(operation *Operation) []string {
	cons := make(map[string]struct{})

	if len(operation.Consumes) == 0 {
		for _, k := range s.spec.Consumes {
			cons[k] = struct{}{}
		}
		return s.structMapKeys(cons)
	}

	for _, c := range operation.Consumes {
		cons[c] = struct{}{}
	}
	return s.structMapKeys(cons)
}

// ProducesFor gets the mediatypes for the operation
func (s *specAnalyzer) ProducesFor(operation *Operation) []string {
	prod := make(map[string]struct{})

	if len(operation.Produces) == 0 {
		for _, k := range s.spec.Produces {
			prod[k] = struct{}{}
		}
		return s.structMapKeys(prod)
	}

	for _, c := range operation.Produces {
		prod[c] = struct{}{}
	}
	return s.structMapKeys(prod)
}

func fieldNameFromParam(param *Parameter) string {
	if nm, ok := param.Extensions.GetString("go-name"); ok {
		return nm
	}
	return swag.ToGoName(param.Name)
}

func (s *specAnalyzer) paramsAsMap(parameters []Parameter, res map[string]Parameter) {
	for _, param := range parameters {
		pr := param
		if pr.Ref.String() != "" {
			obj, _, err := pr.Ref.GetPointer().Get(s.spec)
			if err != nil {
				panic(err)
			}
			pr = obj.(Parameter)
		}
		res[fieldNameFromParam(&pr)] = pr
	}
}

func (s *specAnalyzer) ParametersFor(operationID string) []Parameter {
	gatherParams := func(pi *PathItem, op *Operation) []Parameter {
		bag := make(map[string]Parameter)
		s.paramsAsMap(pi.Parameters, bag)
		s.paramsAsMap(op.Parameters, bag)

		var res []Parameter
		for _, v := range bag {
			res = append(res, v)
		}
		return res
	}
	for _, pi := range s.spec.Paths.Paths {
		if pi.Get != nil && pi.Get.ID == operationID {
			return gatherParams(&pi, pi.Get)
		}
		if pi.Head != nil && pi.Head.ID == operationID {
			return gatherParams(&pi, pi.Head)
		}
		if pi.Options != nil && pi.Options.ID == operationID {
			return gatherParams(&pi, pi.Options)
		}
		if pi.Post != nil && pi.Post.ID == operationID {
			return gatherParams(&pi, pi.Post)
		}
		if pi.Patch != nil && pi.Patch.ID == operationID {
			return gatherParams(&pi, pi.Patch)
		}
		if pi.Put != nil && pi.Put.ID == operationID {
			return gatherParams(&pi, pi.Put)
		}
		if pi.Delete != nil && pi.Delete.ID == operationID {
			return gatherParams(&pi, pi.Delete)
		}
	}
	return nil
}

func (s *specAnalyzer) ParamsFor(method, path string) map[string]Parameter {
	res := make(map[string]Parameter)
	if pi, ok := s.spec.Paths.Paths[path]; ok {
		s.paramsAsMap(pi.Parameters, res)
		s.paramsAsMap(s.operations[strings.ToUpper(method)][path].Parameters, res)
	}
	return res
}

func (s *specAnalyzer) OperationForName(operationID string) (string, string, *Operation, bool) {
	for method, pathItem := range s.operations {
		for path, op := range pathItem {
			if operationID == op.ID {
				return method, path, op, true
			}
		}
	}
	return "", "", nil, false
}

func (s *specAnalyzer) OperationFor(method, path string) (*Operation, bool) {
	if mp, ok := s.operations[strings.ToUpper(method)]; ok {
		op, fn := mp[path]
		return op, fn
	}
	return nil, false
}

func (s *specAnalyzer) Operations() map[string]map[string]*Operation {
	return s.operations
}

func (s *specAnalyzer) structMapKeys(mp map[string]struct{}) []string {
	var result []string
	for k := range mp {
		result = append(result, k)
	}
	return result
}

// AllPaths returns all the paths in the swagger spec
func (s *specAnalyzer) AllPaths() map[string]PathItem {
	if s.spec == nil || s.spec.Paths == nil {
		return nil
	}
	return s.spec.Paths.Paths
}

func (s *specAnalyzer) OperationIDs() []string {
	var result []string
	for method, v := range s.operations {
		for p := range v {
			result = append(result, fmt.Sprintf("%s %s", strings.ToUpper(method), p))
		}
	}
	return result
}

func (s *specAnalyzer) RequiredConsumes() []string {
	return s.structMapKeys(s.consumes)
}

func (s *specAnalyzer) RequiredProduces() []string {
	return s.structMapKeys(s.produces)
}

func (s *specAnalyzer) RequiredSecuritySchemes() []string {
	return s.structMapKeys(s.authSchemes)
}

// SchemaRef is a reference to a schema
type SchemaRef struct {
	Name   string
	Ref    Ref
	Schema *Schema
}

func (s *specAnalyzer) SchemasWithAllOf() (result []SchemaRef) {
	for _, v := range s.allOfs {
		result = append(result, v)
	}
	return
}

func (s *specAnalyzer) AllDefinitions() (result []SchemaRef) {
	for _, v := range s.allSchemas {
		result = append(result, v)
	}
	return
}

func (s *specAnalyzer) AllDefinitionReferences() (result []string) {
	for _, v := range s.references.schemas {
		result = append(result, v.String())
	}
	return
}

func (s *specAnalyzer) AllParameterReferences() (result []string) {
	for _, v := range s.references.parameters {
		result = append(result, v.String())
	}
	return
}

func (s *specAnalyzer) AllResponseReferences() (result []string) {
	for _, v := range s.references.responses {
		result = append(result, v.String())
	}
	return
}
