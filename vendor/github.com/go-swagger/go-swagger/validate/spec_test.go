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

package validate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	intvalidate "github.com/go-swagger/go-swagger/internal/validate"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestIssue52(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "52", "swagger.json")
	jstext, _ := ioutil.ReadFile(fp)

	// as json schema
	var sch spec.Schema
	if assert.NoError(t, json.Unmarshal(jstext, &sch)) {
		validator := intvalidate.NewSchemaValidator(spec.MustLoadSwagger20Schema(), nil, "", strfmt.Default)
		res := validator.Validate(&sch)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".paths in body is required")
	}

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".paths in body is required")
	}

}

func TestIssue53(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "53", "noswagger.json")
	jstext, _ := ioutil.ReadFile(fp)

	// as json schema
	var sch spec.Schema
	if assert.NoError(t, json.Unmarshal(jstext, &sch)) {
		validator := intvalidate.NewSchemaValidator(spec.MustLoadSwagger20Schema(), nil, "", strfmt.Default)
		res := validator.Validate(&sch)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".swagger in body is required")
	}

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		if assert.False(t, res.IsValid()) {
			assert.EqualError(t, res.Errors[0], ".swagger in body is required")
		}
	}
}

func TestIssue62(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "62", "swagger.json")

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(spec.MustLoadSwagger20Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.NotEmpty(t, res.Errors)
		assert.True(t, res.HasErrors())
	}
}

func TestIssue63(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "63", "swagger.json")

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.True(t, res.IsValid())
	}
}

func TestIssue61_MultipleRefs(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "61", "multiple-refs.json")

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.Empty(t, res.Errors)
		assert.True(t, res.IsValid())
	}
}

func TestIssue61_ResolvedRef(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "61", "unresolved-ref-for-name.json")

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.Empty(t, res.Errors)
		assert.True(t, res.IsValid())
	}
}
func TestIssue123(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "123", "swagger.yml")

	// as swagger spec
	doc, err := spec.YAMLSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		for _, e := range res.Errors {
			fmt.Println(e)
		}
		assert.True(t, res.IsValid())
	}
}
