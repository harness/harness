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

var schema = Schema{
	VendorExtensible: VendorExtensible{Extensions: map[string]interface{}{"x-framework": "go-swagger"}},
	SchemaProps: SchemaProps{
		Ref:              MustCreateRef("Cat"),
		Type:             []string{"string"},
		Format:           "date",
		Description:      "the description of this schema",
		Title:            "the title",
		Default:          "blah",
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
		MaxProperties:    int64Ptr(5),
		MinProperties:    int64Ptr(1),
		Required:         []string{"id", "name"},
		Items:            &SchemaOrArray{Schema: &Schema{SchemaProps: SchemaProps{Type: []string{"string"}}}},
		AllOf:            []Schema{Schema{SchemaProps: SchemaProps{Type: []string{"string"}}}},
		Properties: map[string]Schema{
			"id":   Schema{SchemaProps: SchemaProps{Type: []string{"integer"}, Format: "int64"}},
			"name": Schema{SchemaProps: SchemaProps{Type: []string{"string"}}},
		},
		AdditionalProperties: &SchemaOrBool{Allows: true, Schema: &Schema{SchemaProps: SchemaProps{
			Type:   []string{"integer"},
			Format: "int32",
		}}},
	},
	SwaggerSchemaProps: SwaggerSchemaProps{
		Discriminator: "not this",
		ReadOnly:      true,
		XML:           &XMLObject{"sch", "io", "sw", true, true},
		ExternalDocs: &ExternalDocumentation{
			Description: "the documentation etc",
			URL:         "http://readthedocs.org/swagger",
		},
		Example: []interface{}{
			map[string]interface{}{
				"id":   1,
				"name": "a book",
			},
			map[string]interface{}{
				"id":   2,
				"name": "the thing",
			},
		},
	},
}

var schemaJSON = `{
	"x-framework": "go-swagger",
  "$ref": "Cat",
  "description": "the description of this schema",
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
  "title": "the title",
  "default": "blah",
  "maxProperties": 5,
  "minProperties": 1,
  "required": ["id", "name"],
  "items": { 
    "type": "string" 
  },
  "allOf": [
    { 
      "type": "string" 
    }
  ],
  "properties": {
    "id": { 
      "type": "integer",
      "format": "int64"
    },
    "name": {
      "type": "string"
    }
  },
  "discriminator": "not this",
  "readOnly": true,
  "xml": {
    "name": "sch",
    "namespace": "io",
    "prefix": "sw",
    "wrapped": true,
    "attribute": true
  },
  "externalDocs": {
    "description": "the documentation etc",
    "url": "http://readthedocs.org/swagger"
  },
  "example": [
    {
      "id": 1,
      "name": "a book"
    },
    { 
      "id": 2,
      "name": "the thing"
    }
  ],
  "additionalProperties": {
    "type": "integer",
    "format": "int32"
  }
}
`

func TestSchema(t *testing.T) {

	Convey("Schema should", t, func() {

		Convey("serialize", func() {
			expected := map[string]interface{}{}
			json.Unmarshal([]byte(schemaJSON), &expected)
			b, err := json.Marshal(schema)
			So(err, ShouldBeNil)
			var actual map[string]interface{}
			json.Unmarshal(b, &actual)
			So(actual, ShouldBeEquivalentTo, expected)
		})
		Convey("deserialize", func() {
			actual := Schema{}
			err := json.Unmarshal([]byte(schemaJSON), &actual)
			So(err, ShouldBeNil)
			So(actual.Ref, ShouldResemble, schema.Ref)
			So(actual.Description, ShouldEqual, schema.Description)
			So(actual.Maximum, ShouldResemble, schema.Maximum)
			So(actual.Minimum, ShouldResemble, schema.Minimum)
			So(actual.ExclusiveMinimum, ShouldEqual, schema.ExclusiveMinimum)
			So(actual.ExclusiveMaximum, ShouldEqual, schema.ExclusiveMaximum)
			So(actual.MaxLength, ShouldResemble, schema.MaxLength)
			So(actual.MinLength, ShouldResemble, schema.MinLength)
			So(actual.Pattern, ShouldEqual, schema.Pattern)
			So(actual.MaxItems, ShouldResemble, schema.MaxItems)
			So(actual.MinItems, ShouldResemble, schema.MinItems)
			So(actual.UniqueItems, ShouldBeTrue)
			So(actual.MultipleOf, ShouldResemble, schema.MultipleOf)
			So(actual.Enum, ShouldResemble, schema.Enum)
			So(actual.Type, ShouldResemble, schema.Type)
			So(actual.Format, ShouldEqual, schema.Format)
			So(actual.Title, ShouldEqual, schema.Title)
			So(actual.Default, ShouldResemble, schema.Default)
			So(actual.MaxProperties, ShouldResemble, schema.MaxProperties)
			So(actual.MinProperties, ShouldResemble, schema.MinProperties)
			So(actual.Required, ShouldResemble, schema.Required)
			So(actual.Items, ShouldResemble, schema.Items)
			So(actual.AllOf, ShouldResemble, schema.AllOf)
			So(actual.Properties, ShouldResemble, schema.Properties)
			So(actual.Discriminator, ShouldEqual, schema.Discriminator)
			So(actual.ReadOnly, ShouldEqual, schema.ReadOnly)
			So(actual.XML, ShouldResemble, schema.XML)
			So(actual.ExternalDocs, ShouldResemble, schema.ExternalDocs)
			examples := actual.Example.([]interface{})
			expEx := schema.Example.([]interface{})
			ex1 := examples[0].(map[string]interface{})
			ex2 := examples[1].(map[string]interface{})
			exp1 := expEx[0].(map[string]interface{})
			exp2 := expEx[1].(map[string]interface{})
			So(ex1["name"], ShouldEqual, exp1["name"])
			So(ex1["id"], ShouldEqual, exp1["id"])
			So(ex2["name"], ShouldEqual, exp2["name"])
			So(ex2["id"], ShouldEqual, exp2["id"])
			So(actual.AdditionalProperties, ShouldResemble, schema.AdditionalProperties)
			So(actual.Extensions, ShouldBeEquivalentTo, schema.Extensions)
		})
	})

}
