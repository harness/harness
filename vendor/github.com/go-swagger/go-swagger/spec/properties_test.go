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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPropertySerialization(t *testing.T) {

	Convey("Properties should serialize", t, func() {
		Convey("a boolean property", func() {
			prop := BooleanProperty()
			So(prop, ShouldSerializeJSON, `{"type":"boolean"}`)
		})
		Convey("a date property", func() {
			prop := DateProperty()
			So(prop, ShouldSerializeJSON, `{"type":"string","format":"date"}`)
		})
		Convey("a date-time property", func() {
			prop := DateTimeProperty()
			So(prop, ShouldSerializeJSON, `{"type":"string","format":"date-time"}`)
		})
		Convey("a float64 property", func() {
			prop := Float64Property()
			So(prop, ShouldSerializeJSON, `{"type":"number","format":"double"}`)
		})
		Convey("a float32 property", func() {
			prop := Float32Property()
			So(prop, ShouldSerializeJSON, `{"type":"number","format":"float"}`)
		})
		Convey("a int32 property", func() {
			prop := Int32Property()
			So(prop, ShouldSerializeJSON, `{"type":"integer","format":"int32"}`)
		})
		Convey("a int64 property", func() {
			prop := Int64Property()
			So(prop, ShouldSerializeJSON, `{"type":"integer","format":"int64"}`)
		})
		Convey("a string map property", func() {
			prop := MapProperty(StringProperty())
			So(prop, ShouldSerializeJSON, `{"type":"object","additionalProperties":{"type":"string"}}`)
		})
		Convey("an int32 map property", func() {
			prop := MapProperty(Int32Property())
			So(prop, ShouldSerializeJSON, `{"type":"object","additionalProperties":{"type":"integer","format":"int32"}}`)
		})
		Convey("a ref property", func() {
			prop := RefProperty("Dog")
			So(prop, ShouldSerializeJSON, `{"$ref":"Dog"}`)
		})
		Convey("a string property", func() {
			prop := StringProperty()
			So(prop, ShouldSerializeJSON, `{"type":"string"}`)
		})
		Convey("a string property with enums", func() {
			prop := StringProperty()
			prop.Enum = append(prop.Enum, "a", "b")
			So(prop, ShouldSerializeJSON, `{"type":"string","enum":["a","b"]}`)
		})
		Convey("a string array property", func() {
			prop := ArrayProperty(StringProperty())
			So(prop, ShouldSerializeJSON, `{"type":"array","items":{"type":"string"}}`)
		})
	})

	Convey("Properties should deserialize", t, func() {
		Convey("a boolean property", func() {
			prop := BooleanProperty()
			So(`{"type":"boolean"}`, ShouldParseJSON, prop)
		})
		Convey("a date property", func() {
			prop := DateProperty()
			So(`{"format":"date","type":"string"}`, ShouldParseJSON, prop)
		})
		Convey("a date-time property", func() {
			prop := DateTimeProperty()
			So(`{"format":"date-time","type":"string"}`, ShouldParseJSON, prop)
		})
		Convey("a float64 property", func() {
			prop := Float64Property()
			So(`{"format":"double","type":"number"}`, ShouldParseJSON, prop)
		})
		Convey("a float32 property", func() {
			prop := Float32Property()
			So(`{"format":"float","type":"number"}`, ShouldParseJSON, prop)
		})
		Convey("a int32 property", func() {
			prop := Int32Property()
			So(`{"format":"int32","type":"integer"}`, ShouldParseJSON, prop)
		})
		Convey("a int64 property", func() {
			prop := Int64Property()
			So(`{"format":"int64","type":"integer"}`, ShouldParseJSON, prop)
		})
		Convey("a string map property", func() {
			prop := MapProperty(StringProperty())
			So(`{"additionalProperties":{"type":"string"},"type":"object"}`, ShouldParseJSON, prop)
		})
		Convey("an int32 map property", func() {
			prop := MapProperty(Int32Property())
			So(`{"additionalProperties":{"format":"int32","type":"integer"},"type":"object"}`, ShouldParseJSON, prop)
		})
		Convey("a ref property", func() {
			prop := RefProperty("Dog")
			So(`{"$ref":"Dog"}`, ShouldParseJSON, prop)
		})
		Convey("a string property", func() {
			prop := StringProperty()
			So(`{"type":"string"}`, ShouldParseJSON, prop)
		})
		Convey("a string property with enums", func() {
			prop := StringProperty()
			prop.Enum = append(prop.Enum, "a", "b")
			So(`{"enum":["a","b"],"type":"string"}`, ShouldParseJSON, prop)
		})
		Convey("a string array property", func() {
			prop := ArrayProperty(StringProperty())
			So(`{"items":{"type":"string"},"type":"array"}`, ShouldParseJSON, prop)
		})
		Convey("a list of string array properties", func() {
			prop := &Schema{SchemaProps: SchemaProps{
				Items: &SchemaOrArray{Schemas: []Schema{
					Schema{SchemaProps: SchemaProps{Type: []string{"string"}}},
					Schema{SchemaProps: SchemaProps{Type: []string{"string"}}},
				}},
			}}
			So(`{"items":[{"type":"string"},{"type":"string"}]}`, ShouldParseJSON, prop)
		})
	})
}
