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
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/go-swagger/go-swagger/jsonpointer"
	"github.com/go-swagger/go-swagger/swag"
)

// ResolutionCache a cache for resolving urls
type ResolutionCache interface {
	Get(string) (interface{}, bool)
	Set(string, interface{})
}

type simpleCache struct {
	lock  *sync.RWMutex
	store map[string]interface{}
}

func defaultResolutionCache() ResolutionCache {
	return &simpleCache{lock: new(sync.RWMutex), store: map[string]interface{}{
		"http://swagger.io/v2/schema.json":       swaggerSchema,
		"http://json-schema.org/draft-04/schema": jsonSchema,
	}}
}

func (s *simpleCache) Get(uri string) (interface{}, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	v, ok := s.store[uri]
	return v, ok
}

func (s *simpleCache) Set(uri string, data interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.store[uri] = data
}

// ResolveRef resolves a reference against a context root
func ResolveRef(root interface{}, ref *Ref) (*Schema, error) {
	resolver, err := defaultSchemaLoader(root, nil, nil)
	if err != nil {
		return nil, err
	}

	result := new(Schema)
	if err := resolver.Resolve(ref, result); err != nil {
		return nil, err
	}
	return result, nil
}

type schemaLoader struct {
	loadingRef  *Ref
	startingRef *Ref
	currentRef  *Ref
	root        interface{}
	cache       ResolutionCache
	loadDoc     DocLoader
	schemaRef   *Ref
}

var idPtr, _ = jsonpointer.New("/id")
var schemaPtr, _ = jsonpointer.New("/$schema")
var refPtr, _ = jsonpointer.New("/$ref")

func defaultSchemaLoader(root interface{}, ref *Ref, cache ResolutionCache) (*schemaLoader, error) {
	if cache == nil {
		cache = defaultResolutionCache()
	}

	var ptr *jsonpointer.Pointer
	if ref != nil {
		ptr = ref.GetPointer()
	}

	currentRef := nextRef(root, ref, ptr)

	return &schemaLoader{
		root:        root,
		loadingRef:  ref,
		startingRef: ref,
		cache:       cache,
		loadDoc:     swag.JSONDoc,
		currentRef:  currentRef,
	}, nil
}

func idFromNode(node interface{}) (*Ref, error) {
	if idValue, _, err := idPtr.Get(node); err == nil {
		if refStr, ok := idValue.(string); ok && refStr != "" {
			idRef, err := NewRef(refStr)
			if err != nil {
				return nil, err
			}
			return &idRef, nil
		}
	}
	return nil, nil
}

func nextRef(startingNode interface{}, startingRef *Ref, ptr *jsonpointer.Pointer) *Ref {
	if startingRef == nil {
		return nil
	}
	if ptr == nil {
		return startingRef
	}

	ret := startingRef
	var idRef *Ref
	node := startingNode

	for _, tok := range ptr.DecodedTokens() {
		node, _, _ = jsonpointer.GetForToken(node, tok)
		if node == nil {
			break
		}

		idRef, _ = idFromNode(node)
		if idRef != nil {
			nw, err := ret.Inherits(*idRef)
			if err != nil {
				break
			}
			ret = nw
		}

		refRef, _, _ := refPtr.Get(node)
		if refRef != nil {
			rf, _ := NewRef(refRef.(string))
			nw, err := ret.Inherits(rf)
			if err != nil {
				break
			}
			ret = nw
		}

	}
	return ret
}

func (r *schemaLoader) resolveRef(currentRef, ref *Ref, node, target interface{}) error {
	tgt := reflect.ValueOf(target)
	if tgt.Kind() != reflect.Ptr {
		return fmt.Errorf("resolve ref: target needs to be a pointer")
	}

	oldRef := currentRef
	if currentRef != nil {
		var err error
		currentRef, err = currentRef.Inherits(*nextRef(node, ref, currentRef.GetPointer()))
		if err != nil {
			return err
		}
	}
	if currentRef == nil {
		currentRef = ref
	}

	refURL := currentRef.GetURL()
	if refURL == nil {
		return nil
	}
	if currentRef.IsRoot() {
		nv := reflect.ValueOf(node)
		reflect.Indirect(tgt).Set(reflect.Indirect(nv))
		return nil
	}

	if strings.HasPrefix(refURL.String(), "#") {
		res, _, err := ref.GetPointer().Get(node)
		if err != nil {
			res, _, err = ref.GetPointer().Get(r.root)
			if err != nil {
				return err
			}
		}
		rv := reflect.Indirect(reflect.ValueOf(res))
		tgtType := reflect.Indirect(tgt).Type()
		if rv.Type().AssignableTo(tgtType) {
			reflect.Indirect(tgt).Set(reflect.Indirect(reflect.ValueOf(res)))
		} else {
			if err := swag.DynamicJSONToStruct(rv.Interface(), target); err != nil {
				return err
			}
		}

		return nil
	}

	if refURL.Scheme != "" && refURL.Host != "" {
		// most definitely take the red pill
		data, _, _, err := r.load(refURL)

		if ((oldRef == nil && currentRef != nil) ||
			(oldRef != nil && currentRef == nil) ||
			oldRef.String() != currentRef.String()) &&
			((oldRef == nil && ref != nil) ||
				(oldRef != nil && ref == nil) ||
				(oldRef.String() != ref.String())) {

			return r.resolveRef(currentRef, ref, data, target)
		}

		var res interface{}
		if currentRef.String() != "" {
			res, _, err = currentRef.GetPointer().Get(data)
			if err != nil {
				return err
			}
		} else {
			res = data
		}

		if err := swag.DynamicJSONToStruct(res, target); err != nil {
			return err
		}

	}
	return nil
}

func (r *schemaLoader) load(refURL *url.URL) (interface{}, url.URL, bool, error) {
	toFetch := *refURL
	toFetch.Fragment = ""

	data, fromCache := r.cache.Get(toFetch.String())
	if !fromCache {
		b, err := r.loadDoc(toFetch.String())
		if err != nil {
			return nil, url.URL{}, false, err
		}

		if err := json.Unmarshal(b, &data); err != nil {
			return nil, url.URL{}, false, err
		}
		r.cache.Set(toFetch.String(), data)
	}

	return data, toFetch, fromCache, nil
}
func (r *schemaLoader) Resolve(ref *Ref, target interface{}) error {
	if err := r.resolveRef(r.currentRef, ref, r.root, target); err != nil {
		return err
	}

	return nil
}

type specExpander struct {
	spec     *Swagger
	resolver *schemaLoader
}

func expandSpec(spec *Swagger) error {
	resolver, err := defaultSchemaLoader(spec, nil, nil)
	if err != nil {
		return err
	}

	for key, defintition := range spec.Definitions {
		if err := expandSchema(&defintition, resolver); err != nil {
			return err
		}
		spec.Definitions[key] = defintition
	}

	for key, parameter := range spec.Parameters {
		if err := expandParameter(&parameter, resolver); err != nil {
			return err
		}
		spec.Parameters[key] = parameter
	}

	for key, response := range spec.Responses {
		if err := expandResponse(&response, resolver); err != nil {
			return err
		}
		spec.Responses[key] = response
	}

	if spec.Paths != nil {
		for key, path := range spec.Paths.Paths {
			if err := expandPathItem(&path, resolver); err != nil {
				return err
			}
			spec.Paths.Paths[key] = path
		}
	}

	return nil
}

// ExpandSchema expands the refs in the schema object
func ExpandSchema(schema *Schema, root interface{}, cache ResolutionCache) error {

	if schema == nil {
		return nil
	}
	if root == nil {
		root = schema
	}

	nrr, _ := NewRef(schema.ID)
	var rrr *Ref
	if nrr.String() != "" {
		switch root.(type) {
		case *Schema:
			rid, _ := NewRef(root.(*Schema).ID)
			rrr, _ = rid.Inherits(nrr)
		case *Swagger:
			rid, _ := NewRef(root.(*Swagger).ID)
			rrr, _ = rid.Inherits(nrr)
		}

	}

	resolver, err := defaultSchemaLoader(root, rrr, cache)
	if err != nil {
		return err
	}

	if err := expandSchema(schema, resolver); err != nil {
		return nil
	}
	return nil
}

func expandSchema(schema *Schema, resolver *schemaLoader) error {
	if schema == nil {
		return nil
	}

	if schema.Ref.String() == "" && schema.Ref.IsRoot() {
		*schema = *resolver.root.(*Schema)
		return nil
	}
	// create a schema expander and run that
	if schema.Ref.String() != "" {
		currentSchema := *schema
		for currentSchema.Ref.String() != "" {
			var newSchema Schema
			if err := resolver.Resolve(&currentSchema.Ref, &newSchema); err != nil {
				return err
			}
			currentSchema = newSchema
		}
		*schema = currentSchema
	}

	if schema.Items != nil {
		if schema.Items.Schema != nil {
			sch := schema.Items.Schema
			if sch.Ref.String() != "" || sch.Ref.IsRoot() {
				if err := expandSchema(sch, resolver); err != nil {
					return err
				}

			}
		}
		for i := range schema.Items.Schemas {
			sch := &(schema.Items.Schemas[i])
			if sch.Ref.String() != "" || sch.Ref.IsRoot() {
				if err := expandSchema(sch, resolver); err != nil {
					return err
				}

			}
		}
	}
	for i := range schema.AllOf {
		sch := &(schema.AllOf[i])
		if sch.Ref.String() != "" || sch.Ref.IsRoot() {
			if err := expandSchema(sch, resolver); err != nil {
				return err
			}
		}
	}
	for i := range schema.AnyOf {
		sch := &(schema.AnyOf[i])
		if sch.Ref.String() != "" || sch.Ref.IsRoot() {
			if err := expandSchema(sch, resolver); err != nil {
				return err
			}

		}
	}
	for i := range schema.OneOf {
		sch := &(schema.OneOf[i])
		if sch.Ref.String() != "" || sch.Ref.IsRoot() {
			if err := expandSchema(sch, resolver); err != nil {
				return err
			}

		}
	}
	if schema.Not != nil {
		sch := schema.Not
		if sch.Ref.String() != "" || sch.Ref.IsRoot() {
			if err := expandSchema(sch, resolver); err != nil {
				return err
			}

		}
	}
	for k, v := range schema.Properties {
		if v.Ref.String() != "" || v.Ref.IsRoot() {
			if err := expandSchema(&v, resolver); err != nil {
				return err
			}

		}
		schema.Properties[k] = v
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		if err := expandSchema(schema.AdditionalProperties.Schema, resolver); err != nil {
			return err
		}
	}
	for k, v := range schema.PatternProperties {
		vp := &v
		if vp.Ref.String() != "" || v.Ref.IsRoot() {
			if err := expandSchema(&v, resolver); err != nil {
				return err
			}

		}
		schema.PatternProperties[k] = *vp
	}
	for k, v := range schema.Dependencies {
		if v.Schema != nil {
			sch := v.Schema
			if sch.Ref.String() != "" || sch.Ref.IsRoot() {
				if err := expandSchema(sch, resolver); err != nil {
					return err
				}

			}

			schema.Dependencies[k] = v
		}
	}
	if schema.AdditionalItems != nil && schema.AdditionalItems.Schema != nil {
		sch := schema.AdditionalItems.Schema
		if sch.Ref.String() != "" || sch.Ref.IsRoot() {
			if err := expandSchema(sch, resolver); err != nil {
				return err
			}

		}
	}
	for k, v := range schema.Definitions {
		if v.Ref.String() != "" || v.Ref.IsRoot() {
			if err := expandSchema(&v, resolver); err != nil {
				return err
			}

		}
		schema.Definitions[k] = v
	}
	return nil
}

func expandPathItem(pathItem *PathItem, resolver *schemaLoader) error {
	if pathItem == nil {
		return nil
	}
	if err := resolver.Resolve(&pathItem.Ref, &pathItem); err != nil {
		return err
	}

	for idx := range pathItem.Parameters {
		if err := expandParameter(&(pathItem.Parameters[idx]), resolver); err != nil {
			return err
		}
	}
	if err := expandOperation(pathItem.Get, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Head, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Options, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Put, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Post, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Patch, resolver); err != nil {
		return err
	}
	if err := expandOperation(pathItem.Delete, resolver); err != nil {
		return err
	}
	return nil
}

func expandOperation(op *Operation, resolver *schemaLoader) error {
	if op == nil {
		return nil
	}
	for i, param := range op.Parameters {
		if err := expandParameter(&param, resolver); err != nil {
			return err
		}
		op.Parameters[i] = param
	}

	if op.Responses != nil {
		responses := op.Responses
		if err := expandResponse(responses.Default, resolver); err != nil {
			return err
		}
		for code, response := range responses.StatusCodeResponses {
			if err := expandResponse(&response, resolver); err != nil {
				return err
			}
			responses.StatusCodeResponses[code] = response
		}
	}
	return nil
}

func expandResponse(response *Response, resolver *schemaLoader) error {
	if response == nil {
		return nil
	}

	if err := resolver.Resolve(&response.Ref, response); err != nil {
		return err
	}

	if response.Schema != nil {
		if err := expandSchema(response.Schema, resolver); err != nil {
			return err
		}
	}
	return nil
}

func expandParameter(parameter *Parameter, resolver *schemaLoader) error {
	if parameter == nil {
		return nil
	}
	if err := resolver.Resolve(&parameter.Ref, parameter); err != nil {
		return err
	}
	if parameter.Schema != nil {
		if err := expandSchema(parameter.Schema, resolver); err != nil {
			return err
		}

	}
	return nil
}
