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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	testingutil "github.com/go-swagger/go-swagger/internal/testing"
	"github.com/go-swagger/go-swagger/jsonpointer"
	"github.com/go-swagger/go-swagger/swag"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestSpecExpansion(t *testing.T) {
	spec := new(Swagger)
	// resolver, err := defaultSchemaLoader(spec, nil, nil)
	// assert.NoError(t, err)

	err := expandSpec(spec)
	assert.NoError(t, err)

	specDoc, err := swag.JSONDoc("../fixtures/expansion/all-the-things.json")
	assert.NoError(t, err)

	spec = new(Swagger)
	err = json.Unmarshal(specDoc, spec)
	assert.NoError(t, err)

	pet := spec.Definitions["pet"]
	errorModel := spec.Definitions["errorModel"]
	petResponse := spec.Responses["petResponse"]
	petResponse.Schema = &pet
	stringResponse := spec.Responses["stringResponse"]
	tagParam := spec.Parameters["tag"]
	idParam := spec.Parameters["idParam"]

	err = expandSpec(spec)
	assert.NoError(t, err)

	assert.Equal(t, tagParam, spec.Parameters["query"])
	assert.Equal(t, petResponse, spec.Responses["petResponse"])
	assert.Equal(t, petResponse, spec.Responses["anotherPet"])
	assert.Equal(t, pet, *spec.Responses["petResponse"].Schema)
	assert.Equal(t, stringResponse, *spec.Paths.Paths["/"].Get.Responses.Default)
	assert.Equal(t, petResponse, spec.Paths.Paths["/"].Get.Responses.StatusCodeResponses[200])
	assert.Equal(t, pet, *spec.Paths.Paths["/pets"].Get.Responses.StatusCodeResponses[200].Schema.Items.Schema)
	assert.Equal(t, errorModel, *spec.Paths.Paths["/pets"].Get.Responses.Default.Schema)
	assert.Equal(t, pet, spec.Definitions["petInput"].AllOf[0])
	assert.Equal(t, spec.Definitions["petInput"], *spec.Paths.Paths["/pets"].Post.Parameters[0].Schema)
	assert.Equal(t, petResponse, spec.Paths.Paths["/pets"].Post.Responses.StatusCodeResponses[200])
	assert.Equal(t, errorModel, *spec.Paths.Paths["/pets"].Post.Responses.Default.Schema)
	pi := spec.Paths.Paths["/pets/{id}"]
	assert.Equal(t, idParam, pi.Get.Parameters[0])
	assert.Equal(t, petResponse, pi.Get.Responses.StatusCodeResponses[200])
	assert.Equal(t, errorModel, *pi.Get.Responses.Default.Schema)
	assert.Equal(t, idParam, pi.Delete.Parameters[0])
	assert.Equal(t, errorModel, *pi.Delete.Responses.Default.Schema)
}

func TestResponseExpansion(t *testing.T) {
	specDoc, err := swag.JSONDoc("../fixtures/expansion/all-the-things.json")
	assert.NoError(t, err)

	spec := new(Swagger)
	err = json.Unmarshal(specDoc, spec)
	assert.NoError(t, err)

	resolver, err := defaultSchemaLoader(spec, nil, nil)
	assert.NoError(t, err)

	resp := spec.Responses["anotherPet"]
	expected := spec.Responses["petResponse"]

	err = expandResponse(&resp, resolver)
	assert.NoError(t, err)
	assert.Equal(t, expected, resp)

	resp2 := spec.Paths.Paths["/"].Get.Responses.Default
	expected = spec.Responses["stringResponse"]

	err = expandResponse(resp2, resolver)
	assert.NoError(t, err)
	assert.Equal(t, expected, *resp2)

	resp = spec.Paths.Paths["/"].Get.Responses.StatusCodeResponses[200]
	expected = spec.Responses["petResponse"]

	err = expandResponse(&resp, resolver)
	assert.NoError(t, err)
	// assert.Equal(t, expected, resp)
}

func TestParameterExpansion(t *testing.T) {
	paramDoc, err := swag.JSONDoc("../fixtures/expansion/params.json")
	assert.NoError(t, err)

	spec := new(Swagger)
	err = json.Unmarshal(paramDoc, spec)
	assert.NoError(t, err)

	resolver, err := defaultSchemaLoader(spec, nil, nil)
	assert.NoError(t, err)

	param := spec.Parameters["query"]
	expected := spec.Parameters["tag"]

	err = expandParameter(&param, resolver)
	assert.NoError(t, err)
	assert.Equal(t, expected, param)

	param = spec.Paths.Paths["/cars/{id}"].Parameters[0]
	expected = spec.Parameters["id"]

	err = expandParameter(&param, resolver)
	assert.NoError(t, err)
	assert.Equal(t, expected, param)
}

func TestSchemaExpansion(t *testing.T) {
	carsDoc, err := swag.JSONDoc("../fixtures/expansion/schemas1.json")
	assert.NoError(t, err)

	spec := new(Swagger)
	err = json.Unmarshal(carsDoc, spec)
	assert.NoError(t, err)

	resolver, err := defaultSchemaLoader(spec, nil, nil)
	assert.NoError(t, err)

	schema := spec.Definitions["car"]
	oldBrand := schema.Properties["brand"]
	assert.NotEmpty(t, oldBrand.Ref.String())
	assert.NotEqual(t, spec.Definitions["brand"], oldBrand)

	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)

	newBrand := schema.Properties["brand"]
	assert.Empty(t, newBrand.Ref.String())
	assert.Equal(t, spec.Definitions["brand"], newBrand)

	schema = spec.Definitions["truck"]
	assert.NotEmpty(t, schema.Ref.String())

	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.Ref.String())
	assert.Equal(t, spec.Definitions["car"], schema)

	sch := new(Schema)
	err = expandSchema(sch, resolver)
	assert.NoError(t, err)

	schema = spec.Definitions["batch"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.Items.Schema.Ref.String())
	assert.Equal(t, *schema.Items.Schema, spec.Definitions["brand"])

	schema = spec.Definitions["batch2"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.Items.Schemas[0].Ref.String())
	assert.Empty(t, schema.Items.Schemas[1].Ref.String())
	assert.Equal(t, schema.Items.Schemas[0], spec.Definitions["brand"])
	assert.Equal(t, schema.Items.Schemas[1], spec.Definitions["tag"])

	schema = spec.Definitions["allofBoth"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.AllOf[0].Ref.String())
	assert.Empty(t, schema.AllOf[1].Ref.String())
	assert.Equal(t, schema.AllOf[0], spec.Definitions["brand"])
	assert.Equal(t, schema.AllOf[1], spec.Definitions["tag"])

	schema = spec.Definitions["anyofBoth"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.AnyOf[0].Ref.String())
	assert.Empty(t, schema.AnyOf[1].Ref.String())
	assert.Equal(t, schema.AnyOf[0], spec.Definitions["brand"])
	assert.Equal(t, schema.AnyOf[1], spec.Definitions["tag"])

	schema = spec.Definitions["oneofBoth"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.OneOf[0].Ref.String())
	assert.Empty(t, schema.OneOf[1].Ref.String())
	assert.Equal(t, schema.OneOf[0], spec.Definitions["brand"])
	assert.Equal(t, schema.OneOf[1], spec.Definitions["tag"])

	schema = spec.Definitions["notSomething"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.Not.Ref.String())
	assert.Equal(t, *schema.Not, spec.Definitions["tag"])

	schema = spec.Definitions["withAdditional"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.AdditionalProperties.Schema.Ref.String())
	assert.Equal(t, *schema.AdditionalProperties.Schema, spec.Definitions["tag"])

	schema = spec.Definitions["withAdditionalItems"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	assert.Empty(t, schema.AdditionalItems.Schema.Ref.String())
	assert.Equal(t, *schema.AdditionalItems.Schema, spec.Definitions["tag"])

	schema = spec.Definitions["withPattern"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	prop := schema.PatternProperties["^x-ab"]
	assert.Empty(t, prop.Ref.String())
	assert.Equal(t, prop, spec.Definitions["tag"])

	schema = spec.Definitions["deps"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	prop2 := schema.Dependencies["something"]
	assert.Empty(t, prop2.Schema.Ref.String())
	assert.Equal(t, *prop2.Schema, spec.Definitions["tag"])

	schema = spec.Definitions["defined"]
	err = expandSchema(&schema, resolver)
	assert.NoError(t, err)
	prop = schema.Definitions["something"]
	assert.Empty(t, prop.Ref.String())
	assert.Equal(t, prop, spec.Definitions["tag"])

}

func TestDefaultResolutionCache(t *testing.T) {

	cache := defaultResolutionCache()

	sch, ok := cache.Get("not there")
	assert.False(t, ok)
	assert.Nil(t, sch)

	sch, ok = cache.Get("http://swagger.io/v2/schema.json")
	assert.True(t, ok)
	assert.Equal(t, swaggerSchema, sch)

	sch, ok = cache.Get("http://json-schema.org/draft-04/schema")
	assert.True(t, ok)
	assert.Equal(t, jsonSchema, sch)

	cache.Set("something", "here")
	sch, ok = cache.Get("something")
	assert.True(t, ok)
	assert.Equal(t, "here", sch)
}

func resolutionContextServer() *httptest.Server {
	var servedAt string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// fmt.Println("got a request for", req.URL.String())
		if req.URL.Path == "/resolution.json" {

			b, _ := ioutil.ReadFile("../fixtures/specs/resolution.json")
			var ctnt map[string]interface{}
			json.Unmarshal(b, &ctnt)
			ctnt["id"] = servedAt

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			bb, _ := json.Marshal(ctnt)
			rw.Write(bb)
			return
		}
		if req.URL.Path == "/resolution2.json" {
			b, _ := ioutil.ReadFile("../fixtures/specs/resolution2.json")
			var ctnt map[string]interface{}
			json.Unmarshal(b, &ctnt)
			ctnt["id"] = servedAt

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			bb, _ := json.Marshal(ctnt)
			rw.Write(bb)
			return
		}

		if req.URL.Path == "/boolProp.json" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			b, _ := json.Marshal(map[string]interface{}{
				"type": "boolean",
			})
			rw.Write(b)
			return
		}

		if req.URL.Path == "/deeper/stringProp.json" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			b, _ := json.Marshal(map[string]interface{}{
				"type": "string",
			})
			rw.Write(b)
			return
		}

		if req.URL.Path == "/deeper/arrayProp.json" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			b, _ := json.Marshal(map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "file",
				},
			})
			rw.Write(b)
			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	servedAt = server.URL
	return server
}

// func compareSpecs(actual, spec)

func TestResolveRemoteRef(t *testing.T) {
	specs := "../fixtures/specs"
	fileserver := http.FileServer(http.Dir(specs))

	Convey("resolving a remote ref", t, func() {
		server := httptest.NewServer(fileserver)
		Reset(func() {
			server.Close()
		})

		Convey("in a swagger spec", func() {
			rootDoc := new(Swagger)
			b, err := ioutil.ReadFile("../fixtures/specs/refed.json")
			So(err, ShouldBeNil)
			json.Unmarshal(b, rootDoc)

			Convey("resolves root to same schema", func() {
				var result Swagger
				ref, _ := NewRef(server.URL + "/refed.json#")
				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &result)
				So(err, ShouldBeNil)
				compareSpecs(result, *rootDoc)
			})

			Convey("to a schema", func() {

				Convey("from a fragment", func() {
					var tgt Schema
					ref, err := NewRef(server.URL + "/refed.json#/definitions/pet")
					So(err, ShouldBeNil)
					resolver := &schemaLoader{root: rootDoc, cache: defaultResolutionCache(), loadDoc: swag.JSONDoc}
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldBeNil)
					So(tgt.Required, ShouldResemble, []string{"id", "name"})
				})

				Convey("from an invalid fragment", func() {
					var tgt Schema
					ref, err := NewRef(server.URL + "/refed.json#/definitions/NotThere")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldNotBeNil)
				})

				Convey("with a resolution context", func() {
					server.Close()
					server = resolutionContextServer()
					var tgt Schema
					ref, err := NewRef(server.URL + "/resolution.json#/definitions/bool")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldBeNil)
					So(tgt.Type, ShouldResemble, StringOrArray([]string{"boolean"}))
				})

				Convey("with a nested resolution context", func() {
					server.Close()
					server = resolutionContextServer()
					var tgt Schema
					ref, err := NewRef(server.URL + "/resolution.json#/items/items")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldBeNil)
					So(tgt.Type, ShouldResemble, StringOrArray([]string{"string"}))
				})

				Convey("with a nested resolution context with a fragment", func() {
					server.Close()
					server = resolutionContextServer()
					var tgt Schema
					ref, err := NewRef(server.URL + "/resolution2.json#/items/items")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldBeNil)
					So(tgt.Type, ShouldResemble, StringOrArray([]string{"file"}))
				})
			})

			Convey("to a parameter", func() {
				var tgt Parameter
				ref, err := NewRef(server.URL + "/refed.json#/parameters/idParam")
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt.Name, ShouldEqual, "id")
				So(tgt.In, ShouldEqual, "path")
				So(tgt.Description, ShouldEqual, "ID of pet to fetch")
				So(tgt.Required, ShouldBeTrue)
				So(tgt.Type, ShouldEqual, "integer")
				So(tgt.Format, ShouldEqual, "int64")
			})

			Convey("to a path item object", func() {
				var tgt PathItem
				ref, err := NewRef(server.URL + "/refed.json#/paths/" + jsonpointer.Escape("/pets/{id}"))
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt.Get, ShouldResemble, rootDoc.Paths.Paths["/pets/{id}"].Get)
			})

			Convey("to a response object", func() {
				var tgt Response
				ref, err := NewRef(server.URL + "/refed.json#/responses/petResponse")
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt, ShouldResemble, rootDoc.Responses["petResponse"])
			})
		})
	})

}

func TestResolveLocalRef(t *testing.T) {
	rootDoc := new(Swagger)
	json.Unmarshal(testingutil.PetStoreJSONMessage, rootDoc)

	Convey("resolving local a ref", t, func() {

		Convey("in a swagger spec", func() {

			Convey("to a schema", func() {

				Convey("resolves root to same ptr instance", func() {
					result := new(Swagger)
					ref, _ := NewRef("#")
					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err := resolver.Resolve(&ref, result)
					So(err, ShouldBeNil)
					So(result, ShouldResemble, rootDoc)
				})

				Convey("from a fragment", func() {
					var tgt Schema
					ref, err := NewRef("#/definitions/Category")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldBeNil)
					So(tgt.ID, ShouldEqual, "Category")
				})

				Convey("from an invalid fragment", func() {
					var tgt Schema
					ref, err := NewRef("#/definitions/NotThere")
					So(err, ShouldBeNil)

					resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
					err = resolver.Resolve(&ref, &tgt)
					So(err, ShouldNotBeNil)
				})

			})

			Convey("to a parameter", func() {
				rootDoc = new(Swagger)
				b, err := ioutil.ReadFile("../fixtures/specs/refed.json")
				So(err, ShouldBeNil)
				json.Unmarshal(b, rootDoc)

				var tgt Parameter
				ref, err := NewRef("#/parameters/idParam")
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt.Name, ShouldEqual, "id")
				So(tgt.In, ShouldEqual, "path")
				So(tgt.Description, ShouldEqual, "ID of pet to fetch")
				So(tgt.Required, ShouldBeTrue)
				So(tgt.Type, ShouldEqual, "integer")
				So(tgt.Format, ShouldEqual, "int64")
			})

			Convey("to a path item object", func() {
				rootDoc = new(Swagger)
				b, err := ioutil.ReadFile("../fixtures/specs/refed.json")
				So(err, ShouldBeNil)
				json.Unmarshal(b, rootDoc)

				var tgt PathItem
				ref, err := NewRef("#/paths/" + jsonpointer.Escape("/pets/{id}"))
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt.Get, ShouldEqual, rootDoc.Paths.Paths["/pets/{id}"].Get)
			})

			Convey("to a response object", func() {
				rootDoc = new(Swagger)
				b, err := ioutil.ReadFile("../fixtures/specs/refed.json")
				So(err, ShouldBeNil)
				json.Unmarshal(b, rootDoc)

				var tgt Response
				ref, err := NewRef("#/responses/petResponse")
				So(err, ShouldBeNil)

				resolver, _ := defaultSchemaLoader(rootDoc, nil, nil)
				err = resolver.Resolve(&ref, &tgt)
				So(err, ShouldBeNil)
				So(tgt, ShouldResemble, rootDoc.Responses["petResponse"])
			})

		})
	})

}
