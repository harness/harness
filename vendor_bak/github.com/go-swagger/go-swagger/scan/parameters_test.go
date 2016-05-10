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
	goparser "go/parser"
	"log"
	"testing"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/stretchr/testify/assert"
)

func TestScanFileParam(t *testing.T) {
	docFile := "../fixtures/goparsing/classification/operations/noparams.go"
	fileTree, err := goparser.ParseFile(classificationProg.Fset, docFile, nil, goparser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	sp := newParameterParser(classificationProg)
	noParamOps := make(map[string]*spec.Operation)
	err = sp.Parse(fileTree, noParamOps)
	if err != nil {
		log.Fatal(err)
	}
	assert.Len(t, noParamOps, 6)

	of, ok := noParamOps["myOperation"]
	assert.True(t, ok)
	assert.Len(t, of.Parameters, 1)
	fileParam := of.Parameters[0]
	assert.Equal(t, "MyFormFile desc.", fileParam.Description)
	assert.Equal(t, "formData", fileParam.In)
	assert.Equal(t, "file", fileParam.Type)
	assert.False(t, fileParam.Required)
}

func TestParamsParser(t *testing.T) {
	docFile := "../fixtures/goparsing/classification/operations/noparams.go"
	fileTree, err := goparser.ParseFile(classificationProg.Fset, docFile, nil, goparser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	sp := newParameterParser(classificationProg)
	noParamOps := make(map[string]*spec.Operation)
	err = sp.Parse(fileTree, noParamOps)
	if err != nil {
		log.Fatal(err)
	}
	assert.Len(t, noParamOps, 6)

	cr, ok := noParamOps["yetAnotherOperation"]
	assert.True(t, ok)
	assert.Len(t, cr.Parameters, 7)
	for _, param := range cr.Parameters {
		switch param.Name {
		case "id":
			assert.Equal(t, "integer", param.Type)
			assert.Equal(t, "int64", param.Format)
		case "name":
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "", param.Format)
		case "age":
			assert.Equal(t, "integer", param.Type)
			assert.Equal(t, "int32", param.Format)
		case "notes":
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "", param.Format)
		case "extra":
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "", param.Format)
		case "createdAt":
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "date-time", param.Format)
		case "informity":
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "formData", param.In)
		}
	}

	ob, ok := noParamOps["updateOrder"]
	assert.True(t, ok)
	assert.Len(t, ob.Parameters, 1)
	bodyParam := ob.Parameters[0]
	assert.Equal(t, "The order to submit.", bodyParam.Description)
	assert.Equal(t, "body", bodyParam.In)
	assert.Equal(t, "#/definitions/order", bodyParam.Schema.Ref.String())
	assert.True(t, bodyParam.Required)

	op, ok := noParamOps["someOperation"]
	assert.True(t, ok)
	assert.Len(t, op.Parameters, 7)

	for _, param := range op.Parameters {
		switch param.Name {
		case "id":
			assert.Equal(t, "ID of this no model instance.\nids in this application start at 11 and are smaller than 1000", param.Description)
			assert.Equal(t, "path", param.In)
			assert.Equal(t, "integer", param.Type)
			assert.Equal(t, "int64", param.Format)
			assert.True(t, param.Required)
			assert.Equal(t, "ID", param.Extensions["x-go-name"])
			assert.EqualValues(t, 1000, *param.Maximum)
			assert.True(t, param.ExclusiveMaximum)
			assert.EqualValues(t, 10, *param.Minimum)
			assert.True(t, param.ExclusiveMinimum)

		case "score":
			assert.Equal(t, "The Score of this model", param.Description)
			assert.Equal(t, "query", param.In)
			assert.Equal(t, "integer", param.Type)
			assert.Equal(t, "int32", param.Format)
			assert.True(t, param.Required)
			assert.Equal(t, "Score", param.Extensions["x-go-name"])
			assert.EqualValues(t, 45, *param.Maximum)
			assert.False(t, param.ExclusiveMaximum)
			assert.EqualValues(t, 3, *param.Minimum)
			assert.False(t, param.ExclusiveMinimum)

		case "x-hdr-name":
			assert.Equal(t, "Name of this no model instance", param.Description)
			assert.Equal(t, "header", param.In)
			assert.Equal(t, "string", param.Type)
			assert.True(t, param.Required)
			assert.Equal(t, "Name", param.Extensions["x-go-name"])
			assert.EqualValues(t, 4, *param.MinLength)
			assert.EqualValues(t, 50, *param.MaxLength)
			assert.Equal(t, "[A-Za-z0-9-.]*", param.Pattern)

		case "created":
			assert.Equal(t, "Created holds the time when this entry was created", param.Description)
			assert.Equal(t, "query", param.In)
			assert.Equal(t, "string", param.Type)
			assert.Equal(t, "date-time", param.Format)
			assert.False(t, param.Required)
			assert.Equal(t, "Created", param.Extensions["x-go-name"])

		case "foo_slice":
			assert.Equal(t, "a FooSlice has foos which are strings", param.Description)
			assert.Equal(t, "FooSlice", param.Extensions["x-go-name"])
			assert.Equal(t, "query", param.In)
			assert.Equal(t, "array", param.Type)
			assert.False(t, param.Required)
			assert.True(t, param.UniqueItems)
			assert.Equal(t, "pipe", param.CollectionFormat)
			assert.NotNil(t, param.Items, "foo_slice should have had an items property")
			assert.EqualValues(t, 3, *param.MinItems, "'foo_slice' should have had 3 min items")
			assert.EqualValues(t, 10, *param.MaxItems, "'foo_slice' should have had 10 max items")
			itprop := param.Items
			assert.EqualValues(t, 3, *itprop.MinLength, "'foo_slice.items.minLength' should have been 3")
			assert.EqualValues(t, 10, *itprop.MaxLength, "'foo_slice.items.maxLength' should have been 10")
			assert.EqualValues(t, "\\w+", itprop.Pattern, "'foo_slice.items.pattern' should have \\w+")

		case "items":
			assert.Equal(t, "Items", param.Extensions["x-go-name"])
			assert.Equal(t, "body", param.In)
			assert.NotNil(t, param.Schema)
			aprop := param.Schema
			assert.Equal(t, "array", aprop.Type[0])
			assert.NotNil(t, aprop.Items)
			assert.NotNil(t, aprop.Items.Schema)
			itprop := aprop.Items.Schema
			assert.Len(t, itprop.Properties, 4)
			assert.Len(t, itprop.Required, 3)
			assertProperty(t, itprop, "integer", "id", "int32", "ID")
			iprop, ok := itprop.Properties["id"]
			assert.True(t, ok)
			assert.Equal(t, "ID of this no model instance.\nids in this application start at 11 and are smaller than 1000", iprop.Description)
			assert.EqualValues(t, 1000, *iprop.Maximum)
			assert.True(t, iprop.ExclusiveMaximum, "'id' should have had an exclusive maximum")
			assert.NotNil(t, iprop.Minimum)
			assert.EqualValues(t, 10, *iprop.Minimum)
			assert.True(t, iprop.ExclusiveMinimum, "'id' should have had an exclusive minimum")

			assertRef(t, itprop, "pet", "Pet", "#/definitions/pet")
			iprop, ok = itprop.Properties["pet"]
			assert.True(t, ok)
			assert.Equal(t, "The Pet to add to this NoModel items bucket.\nPets can appear more than once in the bucket", iprop.Description)

			assertProperty(t, itprop, "integer", "quantity", "int16", "Quantity")
			iprop, ok = itprop.Properties["quantity"]
			assert.True(t, ok)
			assert.Equal(t, "The amount of pets to add to this bucket.", iprop.Description)
			assert.EqualValues(t, 1, *iprop.Minimum)
			assert.EqualValues(t, 10, *iprop.Maximum)

			assertProperty(t, itprop, "string", "notes", "", "Notes")
			iprop, ok = itprop.Properties["notes"]
			assert.True(t, ok)
			assert.Equal(t, "Notes to add to this item.\nThis can be used to add special instructions.", iprop.Description)

		case "bar_slice":
			assert.Equal(t, "a BarSlice has bars which are strings", param.Description)
			assert.Equal(t, "BarSlice", param.Extensions["x-go-name"])
			assert.Equal(t, "query", param.In)
			assert.Equal(t, "array", param.Type)
			assert.False(t, param.Required)
			assert.True(t, param.UniqueItems)
			assert.Equal(t, "pipe", param.CollectionFormat)
			assert.NotNil(t, param.Items, "bar_slice should have had an items property")
			assert.EqualValues(t, 3, *param.MinItems, "'bar_slice' should have had 3 min items")
			assert.EqualValues(t, 10, *param.MaxItems, "'bar_slice' should have had 10 max items")
			itprop := param.Items
			if assert.NotNil(t, itprop) {
				assert.EqualValues(t, 4, *itprop.MinItems, "'bar_slice.items.minItems' should have been 4")
				assert.EqualValues(t, 9, *itprop.MaxItems, "'bar_slice.items.maxItems' should have been 9")
				itprop2 := itprop.Items
				if assert.NotNil(t, itprop2) {
					assert.EqualValues(t, 5, *itprop2.MinItems, "'bar_slice.items.items.minItems' should have been 5")
					assert.EqualValues(t, 8, *itprop2.MaxItems, "'bar_slice.items.items.maxItems' should have been 8")
					itprop3 := itprop2.Items
					if assert.NotNil(t, itprop3) {
						assert.EqualValues(t, 3, *itprop3.MinLength, "'bar_slice.items.items.items.minLength' should have been 3")
						assert.EqualValues(t, 10, *itprop3.MaxLength, "'bar_slice.items.items.items.maxLength' should have been 10")
						assert.EqualValues(t, "\\w+", itprop3.Pattern, "'bar_slice.items.items.items.pattern' should have \\w+")
					}
				}
			}

		default:
			assert.Fail(t, "unkown property: "+param.Name)
		}
	}
}
