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

func TestParseResponses(t *testing.T) {
	docFile := "../fixtures/goparsing/classification/operations/responses.go"
	fileTree, err := goparser.ParseFile(classificationProg.Fset, docFile, nil, goparser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	rp := newResponseParser(classificationProg)
	responses := make(map[string]spec.Response)
	err = rp.Parse(fileTree, responses)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, responses, 5)
	cr, ok := responses["complexerOne"]
	assert.True(t, ok)
	assert.Len(t, cr.Headers, 6)
	for k, header := range cr.Headers {
		switch k {
		case "id":
			assert.Equal(t, "integer", header.Type)
			assert.Equal(t, "int64", header.Format)
		case "name":
			assert.Equal(t, "string", header.Type)
			assert.Equal(t, "", header.Format)
		case "age":
			assert.Equal(t, "integer", header.Type)
			assert.Equal(t, "int32", header.Format)
		case "notes":
			assert.Equal(t, "string", header.Type)
			assert.Equal(t, "", header.Format)
		case "extra":
			assert.Equal(t, "string", header.Type)
			assert.Equal(t, "", header.Format)
		case "createdAt":
			assert.Equal(t, "string", header.Type)
			assert.Equal(t, "date-time", header.Format)
		}
	}

	res, ok := responses["someResponse"]
	assert.True(t, ok)
	assert.Len(t, res.Headers, 6)

	for k, header := range res.Headers {
		switch k {
		case "id":
			assert.Equal(t, "ID of this some response instance.\nids in this application start at 11 and are smaller than 1000", header.Description)
			assert.Equal(t, "integer", header.Type)
			assert.Equal(t, "int64", header.Format)
			//assert.Equal(t, "ID", header.Extensions["x-go-name"])
			assert.EqualValues(t, 1000, *header.Maximum)
			assert.True(t, header.ExclusiveMaximum)
			assert.EqualValues(t, 10, *header.Minimum)
			assert.True(t, header.ExclusiveMinimum)

		case "score":
			assert.Equal(t, "The Score of this model", header.Description)
			assert.Equal(t, "integer", header.Type)
			assert.Equal(t, "int32", header.Format)
			//assert.Equal(t, "Score", header.Extensions["x-go-name"])
			assert.EqualValues(t, 45, *header.Maximum)
			assert.False(t, header.ExclusiveMaximum)
			assert.EqualValues(t, 3, *header.Minimum)
			assert.False(t, header.ExclusiveMinimum)

		case "x-hdr-name":
			assert.Equal(t, "Name of this some response instance", header.Description)
			assert.Equal(t, "string", header.Type)
			//assert.Equal(t, "Name", header.Extensions["x-go-name"])
			assert.EqualValues(t, 4, *header.MinLength)
			assert.EqualValues(t, 50, *header.MaxLength)
			assert.Equal(t, "[A-Za-z0-9-.]*", header.Pattern)

		case "created":
			assert.Equal(t, "Created holds the time when this entry was created", header.Description)
			assert.Equal(t, "string", header.Type)
			assert.Equal(t, "date-time", header.Format)
			//assert.Equal(t, "Created", header.Extensions["x-go-name"])

		case "foo_slice":
			assert.Equal(t, "a FooSlice has foos which are strings", header.Description)
			//assert.Equal(t, "FooSlice", header.Extensions["x-go-name"])
			assert.Equal(t, "array", header.Type)
			assert.True(t, header.UniqueItems)
			assert.Equal(t, "pipe", header.CollectionFormat)
			assert.NotNil(t, header.Items, "foo_slice should have had an items property")
			assert.EqualValues(t, 3, *header.MinItems, "'foo_slice' should have had 3 min items")
			assert.EqualValues(t, 10, *header.MaxItems, "'foo_slice' should have had 10 max items")
			itprop := header.Items
			assert.EqualValues(t, 3, *itprop.MinLength, "'foo_slice.items.minLength' should have been 3")
			assert.EqualValues(t, 10, *itprop.MaxLength, "'foo_slice.items.maxLength' should have been 10")
			assert.EqualValues(t, "\\w+", itprop.Pattern, "'foo_slice.items.pattern' should have \\w+")

		case "bar_slice":
			assert.Equal(t, "a BarSlice has bars which are strings", header.Description)
			assert.Equal(t, "array", header.Type)
			assert.True(t, header.UniqueItems)
			assert.Equal(t, "pipe", header.CollectionFormat)
			assert.NotNil(t, header.Items, "bar_slice should have had an items property")
			assert.EqualValues(t, 3, *header.MinItems, "'bar_slice' should have had 3 min items")
			assert.EqualValues(t, 10, *header.MaxItems, "'bar_slice' should have had 10 max items")
			itprop := header.Items
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
			assert.Fail(t, "unkown property: "+k)
		}
	}

	assert.NotNil(t, res.Schema)
	aprop := res.Schema
	assert.Equal(t, "array", aprop.Type[0])
	assert.NotNil(t, aprop.Items)
	assert.NotNil(t, aprop.Items.Schema)
	itprop := aprop.Items.Schema
	assert.Len(t, itprop.Properties, 4)
	assert.Len(t, itprop.Required, 3)
	assertProperty(t, itprop, "integer", "id", "int32", "ID")
	iprop, ok := itprop.Properties["id"]
	assert.True(t, ok)
	assert.Equal(t, "ID of this some response instance.\nids in this application start at 11 and are smaller than 1000", iprop.Description)
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

	res, ok = responses["resp"]
	assert.True(t, ok)
	assert.NotNil(t, res.Schema)
	assert.Equal(t, "#/definitions/user", res.Schema.Ref.String())
}
