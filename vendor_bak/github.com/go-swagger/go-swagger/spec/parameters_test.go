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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var parameter = Parameter{
	VendorExtensible: VendorExtensible{Extensions: map[string]interface{}{
		"x-framework": "swagger-go",
	}},
	Refable: Refable{Ref: MustCreateRef("Dog")},
	CommonValidations: CommonValidations{
		Maximum:          float64Ptr(100),
		ExclusiveMaximum: true,
		ExclusiveMinimum: true,
		Minimum:          float64Ptr(5),
		MaxLength:        int64Ptr(100),
		MinLength:        int64Ptr(5),
		Pattern:          "\\w{1,5}\\w+",
		MaxItems:         int64Ptr(100),
		MinItems:         int64Ptr(5),
		UniqueItems:      true,
		MultipleOf:       float64Ptr(5),
		Enum:             []interface{}{"hello", "world"},
	},
	SimpleSchema: SimpleSchema{
		Type:             "string",
		Format:           "date",
		CollectionFormat: "csv",
		Items: &Items{
			Refable: Refable{Ref: MustCreateRef("Cat")},
		},
		Default: "8",
	},
	ParamProps: ParamProps{
		Name:        "param-name",
		In:          "header",
		Required:    true,
		Schema:      &Schema{SchemaProps: SchemaProps{Type: []string{"string"}}},
		Description: "the description of this parameter",
	},
}

var parameterJSON = `{
	"items": { 
		"$ref": "Cat"
	},
	"x-framework": "swagger-go",
  "$ref": "Dog",
  "description": "the description of this parameter",
  "maximum": 100,
  "minimum": 5,
  "exclusiveMaximum": true,
  "exclusiveMinimum": true,
  "maxLength": 100,
  "minLength": 5,
  "pattern": "\\w{1,5}\\w+",
  "maxItems": 100,
  "minItems": 5,
  "uniqueItems": true,
  "multipleOf": 5,
  "enum": ["hello", "world"],
  "type": "string",
  "format": "date",
	"name": "param-name",
	"in": "header",
	"required": true,
	"schema": {
		"type": "string"
	},
	"collectionFormat": "csv",
	"default": "8"
}`

func TestIntegrationParameter(t *testing.T) {
	Convey("for all properties a parameter should", t, func() {
		Convey("serialize", func() {
			expected := map[string]interface{}{}
			json.Unmarshal([]byte(parameterJSON), &expected)
			b, err := json.Marshal(parameter)
			So(err, ShouldBeNil)
			var actual map[string]interface{}
			err = json.Unmarshal(b, &actual)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("deserialize", func() {
			actual := Parameter{}
			err := json.Unmarshal([]byte(parameterJSON), &actual)
			So(err, ShouldBeNil)
			So(actual.Items, ShouldResemble, parameter.Items)
			So(actual.Extensions, ShouldResemble, parameter.Extensions)
			So(actual.Ref, ShouldResemble, parameter.Ref)
			So(actual.Description, ShouldEqual, parameter.Description)
			So(actual.Maximum, ShouldResemble, parameter.Maximum)
			So(actual.Minimum, ShouldResemble, parameter.Minimum)
			So(actual.ExclusiveMinimum, ShouldEqual, parameter.ExclusiveMinimum)
			So(actual.ExclusiveMaximum, ShouldEqual, parameter.ExclusiveMaximum)
			So(actual.MaxLength, ShouldResemble, parameter.MaxLength)
			So(actual.MinLength, ShouldResemble, parameter.MinLength)
			So(actual.Pattern, ShouldEqual, parameter.Pattern)
			So(actual.MaxItems, ShouldResemble, parameter.MaxItems)
			So(actual.MinItems, ShouldResemble, parameter.MinItems)
			So(actual.UniqueItems, ShouldBeTrue)
			So(actual.MultipleOf, ShouldResemble, parameter.MultipleOf)
			So(actual.Enum, ShouldResemble, parameter.Enum)
			So(actual.Type, ShouldResemble, parameter.Type)
			So(actual.Format, ShouldEqual, parameter.Format)
			So(actual.Name, ShouldEqual, parameter.Name)
			So(actual.In, ShouldEqual, parameter.In)
			So(actual.Required, ShouldEqual, parameter.Required)
			So(actual.Schema, ShouldResemble, parameter.Schema)
			So(actual.CollectionFormat, ShouldEqual, parameter.CollectionFormat)
			So(actual.Default, ShouldResemble, parameter.Default)
		})
	})
}

func TestParameterSerialization(t *testing.T) {

	Convey("Parameters should serialize", t, func() {
		items := &Items{
			SimpleSchema: SimpleSchema{Type: "string"},
		}
		Convey("a query parameter", func() {
			param := QueryParam("")
			param.Type = "string"
			So(param, ShouldSerializeJSON, `{"type":"string","in":"query"}`)
		})

		Convey("a query parameter with array", func() {

			param := QueryParam("").CollectionOf(items, "multi")
			So(param, ShouldSerializeJSON, `{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"query"}`)
		})

		Convey("a path parameter", func() {
			param := PathParam("").Typed("string", "")
			So(param, ShouldSerializeJSON, `{"type":"string","in":"path","required":true}`)
		})

		Convey("a path parameter with string array", func() {
			param := PathParam("").CollectionOf(items, "multi")

			So(param, ShouldSerializeJSON, `{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"path","required":true}`)
		})

		Convey("a path parameter with an int array", func() {
			items = &Items{
				SimpleSchema: SimpleSchema{Type: "int", Format: "int32"},
			}
			param := PathParam("").CollectionOf(items, "multi")
			So(param, ShouldSerializeJSON, `{"type":"array","items":{"type":"int","format":"int32"},"collectionFormat":"multi","in":"path","required":true}`)
		})

		Convey("a header parameter", func() {
			param := HeaderParam("").Typed("string", "")
			So(param, ShouldSerializeJSON, `{"type":"string","in":"header","required":true}`)
		})

		Convey("a header parameter with string array", func() {
			param := HeaderParam("").CollectionOf(items, "multi")
			So(param, ShouldSerializeJSON, `{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"header","required":true}`)
		})

		Convey("a body parameter", func() {
			schema := &Schema{SchemaProps: SchemaProps{
				Properties: map[string]Schema{
					"name": Schema{SchemaProps: SchemaProps{
						Type: []string{"string"},
					}},
				},
			}}
			param := BodyParam("", schema)
			So(param, ShouldSerializeJSON, `{"type":"object","in":"body","schema":{"properties":{"name":{"type":"string"}}}}`)
		})

		Convey("a ref body parameter", func() {
			schema := &Schema{
				SchemaProps: SchemaProps{Ref: MustCreateRef("Cat")},
			}
			param := BodyParam("", schema)
			So(param, ShouldSerializeJSON, `{"type":"object","in":"body","schema":{"$ref":"Cat"}}`)
		})

		Convey("serialize an array body parameter", func() {
			param := BodyParam("", ArrayProperty(RefProperty("Cat")))
			So(param, ShouldSerializeJSON, `{"type":"object","in":"body","schema":{"type":"array","items":{"$ref":"Cat"}}}`)
		})
	})
}
