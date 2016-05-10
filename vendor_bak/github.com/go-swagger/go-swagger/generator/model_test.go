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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/swag"
	"github.com/stretchr/testify/assert"
)

type templateTest struct {
	t        testing.TB
	template *template.Template
}

func (tt *templateTest) assertRender(data interface{}, expected string) bool {
	buf := bytes.NewBuffer(nil)
	err := tt.template.Execute(buf, data)
	if !assert.NoError(tt.t, err) {
		return false
	}
	return assert.Equal(tt.t, expected, buf.String())
}

func TestGenerateModel_Sanity(t *testing.T) {
	// just checks if it can render and format these things
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions

		//k := "Comment"
		//schema := definitions[k]
		for k, schema := range definitions {
			genModel, err := makeGenDefinition(k, "models", schema, specDoc)

			// log.Printf("trying model: %s", k)
			if assert.NoError(t, err) {
				//b, _ := json.MarshalIndent(genModel, "", "  ")
				//fmt.Println(string(b))
				rendered := bytes.NewBuffer(nil)

				err := modelTemplate.Execute(rendered, genModel)
				if assert.NoError(t, err) {
					if assert.NoError(t, err) {
						_, err := formatGoFile(strings.ToLower(k)+".go", rendered.Bytes())
						assert.NoError(t, err)
						//if assert.NoError(t, err) {
						//fmt.Println(string(formatted))
						//} else {
						//fmt.Println(rendered.String())
						////break
						//}

						//assert.EqualValues(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(formatted)))
					}
				}
			}
		}
	}
}

func TestGenerateModel_DocString(t *testing.T) {
	templ := template.Must(template.New("docstring").Funcs(FuncMap).Parse(string(assets["docstring.gotmpl"])))
	tt := templateTest{t, templ}

	var gmp GenSchema
	gmp.Title = "The title of the property"
	gmp.Description = "The description of the property"
	var expected = `The title of the property

The description of the property
`
	tt.assertRender(gmp, expected)

	gmp.Title = ""
	expected = `The description of the property
`
	tt.assertRender(gmp, expected)

	gmp.Description = ""
	gmp.Name = "theModel"
	expected = `TheModel the model
`
	tt.assertRender(gmp, expected)
}

func TestGenerateModel_PropertyValidation(t *testing.T) {
	templ := template.Must(template.New("propertyValidationDocString").Funcs(FuncMap).Parse(string(assets["validation/structfield.gotmpl"])))
	tt := templateTest{t, templ}

	var gmp GenSchema
	gmp.Required = true
	tt.assertRender(gmp, `
Required: true
`)
	var fl float64 = 10
	var in1 int64 = 20
	var in2 int64 = 30
	gmp.Maximum = &fl
	gmp.ExclusiveMaximum = true
	gmp.Minimum = &fl
	gmp.ExclusiveMinimum = true
	gmp.MaxLength = &in1
	gmp.MinLength = &in1
	gmp.Pattern = "\\w[\\w- ]+"
	gmp.MaxItems = &in2
	gmp.MinItems = &in2
	gmp.UniqueItems = true

	tt.assertRender(gmp, `
Required: true
Maximum: < 10
Minimum: > 10
Max Length: 20
Min Length: 20
Pattern: \w[\w- ]+
Max Items: 30
Min Items: 30
Unique: true
`)

	gmp.Required = false
	gmp.ExclusiveMaximum = false
	gmp.ExclusiveMinimum = false
	tt.assertRender(gmp, `
Maximum: 10
Minimum: 10
Max Length: 20
Min Length: 20
Pattern: \w[\w- ]+
Max Items: 30
Min Items: 30
Unique: true
`)

}

func TestGenerateModel_SchemaField(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("structfield")}

	var gmp GenSchema
	gmp.Name = "some name"
	gmp.resolvedType = resolvedType{GoType: "string", IsPrimitive: true}
	gmp.Title = "The title of the property"

	tt.assertRender(gmp, `/* The title of the property
 */
`+"SomeName string `json:\"some name,omitempty\"`\n")

	var fl float64 = 10
	var in1 int64 = 20
	var in2 int64 = 30

	gmp.Description = "The description of the property"
	gmp.Required = true
	gmp.Maximum = &fl
	gmp.ExclusiveMaximum = true
	gmp.Minimum = &fl
	gmp.ExclusiveMinimum = true
	gmp.MaxLength = &in1
	gmp.MinLength = &in1
	gmp.Pattern = "\\w[\\w- ]+"
	gmp.MaxItems = &in2
	gmp.MinItems = &in2
	gmp.UniqueItems = true
	gmp.ReadOnly = true
	tt.assertRender(gmp, `/* The title of the property

The description of the property

Required: true
Read Only: true
Maximum: < 10
Minimum: > 10
Max Length: 20
Min Length: 20
Pattern: \w[\w- ]+
Max Items: 30
Min Items: 30
Unique: true
 */
`+"SomeName string `json:\"some name,omitempty\"`\n")
}

var schTypeGenDataSimple = []struct {
	Value    GenSchema
	Expected string
}{
	{GenSchema{resolvedType: resolvedType{GoType: "string", IsPrimitive: true}}, "string"},
	{GenSchema{resolvedType: resolvedType{GoType: "string", IsPrimitive: true, IsNullable: true}}, "*string"},
	{GenSchema{resolvedType: resolvedType{GoType: "bool", IsPrimitive: true}}, "bool"},
	{GenSchema{resolvedType: resolvedType{GoType: "int32", IsPrimitive: true}}, "int32"},
	{GenSchema{resolvedType: resolvedType{GoType: "int64", IsPrimitive: true}}, "int64"},
	{GenSchema{resolvedType: resolvedType{GoType: "float32", IsPrimitive: true}}, "float32"},
	{GenSchema{resolvedType: resolvedType{GoType: "float64", IsPrimitive: true}}, "float64"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Base64", IsPrimitive: true}}, "strfmt.Base64"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Date", IsPrimitive: true}}, "strfmt.Date"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.DateTime", IsPrimitive: true}}, "strfmt.DateTime"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.URI", IsPrimitive: true}}, "strfmt.URI"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Email", IsPrimitive: true}}, "strfmt.Email"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Hostname", IsPrimitive: true}}, "strfmt.Hostname"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.IPv4", IsPrimitive: true}}, "strfmt.IPv4"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.IPv6", IsPrimitive: true}}, "strfmt.IPv6"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.UUID", IsPrimitive: true}}, "strfmt.UUID"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.UUID3", IsPrimitive: true}}, "strfmt.UUID3"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.UUID4", IsPrimitive: true}}, "strfmt.UUID4"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.UUID5", IsPrimitive: true}}, "strfmt.UUID5"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.ISBN", IsPrimitive: true}}, "strfmt.ISBN"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.ISBN10", IsPrimitive: true}}, "strfmt.ISBN10"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.ISBN13", IsPrimitive: true}}, "strfmt.ISBN13"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.CreditCard", IsPrimitive: true}}, "strfmt.CreditCard"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.SSN", IsPrimitive: true}}, "strfmt.SSN"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.HexColor", IsPrimitive: true}}, "strfmt.HexColor"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.RGBColor", IsPrimitive: true}}, "strfmt.RGBColor"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Duration", IsPrimitive: true}}, "strfmt.Duration"},
	{GenSchema{resolvedType: resolvedType{GoType: "strfmt.Password", IsPrimitive: true}}, "strfmt.Password"},
	{GenSchema{resolvedType: resolvedType{GoType: "interface{}", IsInterface: true}}, "interface{}"},
	{GenSchema{resolvedType: resolvedType{GoType: "[]int32", IsArray: true}}, "[]int32"},
	{GenSchema{resolvedType: resolvedType{GoType: "[]string", IsArray: true}}, "[]string"},
	{GenSchema{resolvedType: resolvedType{GoType: "map[string]int32", IsMap: true}}, "map[string]int32"},
	{GenSchema{resolvedType: resolvedType{GoType: "models.Task", IsComplexObject: true, IsNullable: true, IsAnonymous: false}}, "*models.Task"},
}

func TestGenSchemaType(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schemaType")}
	for _, v := range schTypeGenDataSimple {
		tt.assertRender(v.Value, v.Expected)
	}
}
func TestGenerateModel_Primitives(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schema")}
	for _, v := range schTypeGenDataSimple {
		val := v.Value
		val.ReceiverName = "o"
		if val.IsComplexObject {
			continue
		}
		val.Name = "theType"
		exp := v.Expected
		if val.IsInterface {
			tt.assertRender(val, "type TheType "+exp+"\n\n")
			continue
		}
		tt.assertRender(val, "type TheType "+exp+"\n// Validate validates this the type\nfunc (o theType) Validate(formats strfmt.Registry) error {\n  return nil\n}\n")
	}
}

func TestGenerateModel_Nota(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Nota"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type Nota map[string]int32", res)
			}
		}
	}
}

func TestGenerateModel_NotaWithRef(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NotaWithRef"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("nota_with_ref.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type NotaWithRef map[string]*Notable", res)
				}
			}
		}
	}
}

func TestGenerateModel_NotaWithMeta(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NotaWithMeta"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("nota_with_meta.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type NotaWithMeta map[string]*NotaWithMetaAnon", res)
					assertInCode(t, "type NotaWithMetaAnon struct {", res)
					assertInCode(t, "Comment string `json:\"comment,omitempty\"`", res)
					assertInCode(t, "Count *int32 `json:\"count,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_RunParameters(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "RunParameters"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.IsAdditionalProperties)
			assert.True(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsMap)
			assert.False(t, genModel.IsAnonymous)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, "BranchName *string `json:\"branch_name,omitempty\"`", res)
				assertInCode(t, "CommitSha *string `json:\"commit_sha,omitempty\"`", res)
				assertInCode(t, "Refs interface{} `json:\"refs,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_NotaWithName(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NotaWithName"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.True(t, genModel.IsAdditionalProperties)
			assert.False(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsMap)
			assert.False(t, genModel.IsAnonymous)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, k+" map[string]int32 `json:\"-\"`", res)
				assertInCode(t, "Name string `json:\"name,omitempty\"`", res)
				assertInCode(t, k+") UnmarshalJSON", res)
				assertInCode(t, k+") MarshalJSON", res)
				assertInCode(t, "json.Marshal(m)", res)
				assertInCode(t, "json.Marshal(m."+k+")", res)
				assertInCode(t, "json.Unmarshal(data, &stage1)", res)
				assertInCode(t, "json.Unmarshal(data, &stage2)", res)
				assertInCode(t, "json.Unmarshal(v, &toadd)", res)
				assertInCode(t, "result[k] = toadd", res)
				assertInCode(t, "m."+k+" = result", res)
				for _, p := range genModel.Properties {
					assertInCode(t, "delete(stage2, \""+p.Name+"\")", res)
				}

			}
		}
	}
}

func TestGenerateModel_NotaWithRefRegistry(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NotaWithRefRegistry"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("nota_with_ref_registry.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type "+k+" map[string]map[string]map[string]*Notable", res)
				}
			}
		}
	}
}

func TestGenerateModel_NotaWithMetaRegistry(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NotaWithMetaRegistry"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("nota_with_meta_registry.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type "+k+" map[string]map[string]map[string]*NotaWithMetaRegistryAnon", res)
					assertInCode(t, "type NotaWithMetaRegistryAnon struct {", res)
					assertInCode(t, "Comment string `json:\"comment,omitempty\"`", res)
					assertInCode(t, "Count *int32 `json:\"count,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithMap(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithMap"]
		genModel, err := makeGenDefinition("WithMap", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type WithMap struct {", res)
				assertInCode(t, "Data map[string]string `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapInterface(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithMapInterface"]
		genModel, err := makeGenDefinition("WithMapInterface", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "extraInfo")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			assert.Equal(t, "map[string]interface{}", prop.GoType)
			assert.True(t, prop.Required)
			assert.True(t, prop.HasValidations)
			assert.False(t, prop.NeedsValidation)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				//fmt.Println(res)
				assertInCode(t, "type WithMapInterface struct {", res)
				assertInCode(t, "ExtraInfo map[string]interface{} `json:\"extraInfo,omitempty\"`", res)
				assertInCode(t, "ExtraInfo map[string]interface{} `json:\"extraInfo,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapRef(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithMapRef"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, "Data map[string]*Notable `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapComplex(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithMapComplex"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, "Data map[string]*"+k+"DataAnon `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapRegistry(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithMapRegistry"]
		genModel, err := makeGenDefinition("WithMap", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type WithMap struct {", res)
				assertInCode(t, "Data map[string]map[string]map[string]string `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapRegistryRef(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithMapRegistryRef"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, "Data map[string]map[string]map[string]*Notable `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithMapComplexRegistry(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithMapComplexRegistry"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.False(t, genModel.HasAdditionalProperties)
			prop := getDefinitionProperty(genModel, "data")
			assert.True(t, prop.HasAdditionalProperties)
			assert.True(t, prop.IsMap)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type "+k+" struct {", res)
				assertInCode(t, "Data map[string]map[string]map[string]*"+k+"DataAnon `json:\"data,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithAdditional(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithAdditional"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.NotEmpty(t, genModel.ExtraSchemas) {
			assert.False(t, genModel.HasAdditionalProperties)
			assert.False(t, genModel.IsMap)
			assert.False(t, genModel.IsAdditionalProperties)
			assert.True(t, genModel.IsComplexObject)

			sch := genModel.ExtraSchemas[0]
			assert.True(t, sch.HasAdditionalProperties)
			assert.False(t, sch.IsMap)
			assert.True(t, sch.IsAdditionalProperties)
			assert.False(t, sch.IsComplexObject)

			if assert.NotNil(t, sch.AdditionalProperties) {
				prop := findProperty(genModel.Properties, "data")
				assert.False(t, prop.HasAdditionalProperties)
				assert.False(t, prop.IsMap)
				assert.False(t, prop.IsAdditionalProperties)
				assert.True(t, prop.IsComplexObject)
				buf := bytes.NewBuffer(nil)
				err := modelTemplate.Execute(buf, genModel)
				if assert.NoError(t, err) {
					res := buf.String()
					assertInCode(t, "type "+k+" struct {", res)
					assertInCode(t, "Data *"+k+"Data `json:\"data,omitempty\"`", res)
					assertInCode(t, "type "+k+"Data struct {", res)
					assertInCode(t, k+"Data map[string]string `json:\"-\"`", res)
					assertInCode(t, "Name string `json:\"name,omitempty\"`", res)
					assertInCode(t, k+"Data) UnmarshalJSON", res)
					assertInCode(t, k+"Data) MarshalJSON", res)
					assertInCode(t, "json.Marshal(m)", res)
					assertInCode(t, "json.Marshal(m."+k+"Data)", res)
					assertInCode(t, "json.Unmarshal(data, &stage1)", res)
					assertInCode(t, "json.Unmarshal(data, &stage2)", res)
					assertInCode(t, "json.Unmarshal(v, &toadd)", res)
					assertInCode(t, "result[k] = toadd", res)
					assertInCode(t, "m."+k+"Data = result", res)
					for _, p := range sch.Properties {
						assertInCode(t, "delete(stage2, \""+p.Name+"\")", res)
					}
				}
			}
		}
	}
}

func TestGenerateModel_JustRef(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schema")}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["JustRef"]
		genModel, err := makeGenDefinition("JustRef", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, genModel.AllOf)
			assert.True(t, genModel.IsComplexObject)
			assert.Equal(t, "JustRef", genModel.Name)
			assert.Equal(t, "JustRef", genModel.GoType)
			buf := bytes.NewBuffer(nil)
			tt.template.Execute(buf, genModel)
			res := buf.String()
			assertInCode(t, "type JustRef struct {", res)
			assertInCode(t, "Notable", res)
		}
	}
}

func TestGenerateModel_WithRef(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schema")}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithRef"]
		genModel, err := makeGenDefinition("WithRef", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.True(t, genModel.IsComplexObject)
			assert.Equal(t, "WithRef", genModel.Name)
			assert.Equal(t, "WithRef", genModel.GoType)
			buf := bytes.NewBuffer(nil)
			tt.template.Execute(buf, genModel)
			res := buf.String()
			assertInCode(t, "type WithRef struct {", res)
			assertInCode(t, "Notes *Notable `json:\"notes,omitempty\"`", res)
		}
	}
}

func TestGenerateModel_WithNullableRef(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schema")}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithNullableRef"]
		genModel, err := makeGenDefinition("WithNullableRef", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.True(t, genModel.IsComplexObject)
			assert.Equal(t, "WithNullableRef", genModel.Name)
			assert.Equal(t, "WithNullableRef", genModel.GoType)
			prop := getDefinitionProperty(genModel, "notes")
			assert.True(t, prop.IsNullable)
			assert.True(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			tt.template.Execute(buf, genModel)
			res := buf.String()
			assertInCode(t, "type WithNullableRef struct {", res)
			assertInCode(t, "Notes *Notable `json:\"notes,omitempty\"`", res)
		}
	}
}

func TestGenerateModel_Scores(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Scores"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("scores.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type Scores []float32", res)
				}
			}
		}
	}
}

func TestGenerateModel_JaggedScores(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "JaggedScores"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("jagged_scores.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type JaggedScores [][][]float32", res)
				}
			}
		}
	}
}

func TestGenerateModel_Notables(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Notables"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.Equal(t, "[]*Notable", genModel.GoType) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("notables.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type Notables []*Notable", res)
				}
			}
		}
	}
}

func TestGenerateModel_Notablix(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Notablix"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("notablix.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type Notablix [][][]*Notable", res)
				}
			}
		}
	}
}

func TestGenerateModel_Stats(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Stats"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("stats.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type Stats []*StatsItems0", res)
					assertInCode(t, "type StatsItems0 struct {", res)
					assertInCode(t, "Points []int64 `json:\"points,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_Statix(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Statix"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("statix.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "type Statix [][][]*StatixItems0", res)
					assertInCode(t, "type StatixItems0 struct {", res)
					assertInCode(t, "Points []int64 `json:\"points,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithItems(t *testing.T) {
	tt := templateTest{t, modelTemplate.Lookup("schema")}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithItems"]
		genModel, err := makeGenDefinition("WithItems", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Nil(t, genModel.Items)
			assert.True(t, genModel.IsComplexObject)
			prop := getDefinitionProperty(genModel, "tags")
			assert.NotNil(t, prop.Items)
			assert.True(t, prop.IsArray)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := tt.template.Execute(buf, genModel)
			if assert.NoError(t, err) {
				res := buf.String()
				assertInCode(t, "type WithItems struct {", res)
				assertInCode(t, "Tags []string `json:\"tags,omitempty\"`", res)
			}
		}
	}
}

func TestGenerateModel_WithComplexItems(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithComplexItems"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Nil(t, genModel.Items)
			assert.True(t, genModel.IsComplexObject)
			prop := getDefinitionProperty(genModel, "tags")
			assert.NotNil(t, prop.Items)
			assert.True(t, prop.IsArray)
			assert.False(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				b, err := formatGoFile("with_complex_items.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(b)
					assertInCode(t, "type WithComplexItems struct {", res)
					assertInCode(t, "type WithComplexItemsTagsItems0 struct {", res)
					assertInCode(t, "Tags []*WithComplexItemsTagsItems0 `json:\"tags,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithItemsAndAdditional(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithItemsAndAdditional"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Nil(t, genModel.Items)
			assert.True(t, genModel.IsComplexObject)
			prop := getDefinitionProperty(genModel, "tags")
			assert.True(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				b, err := formatGoFile("with_complex_items.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(b)
					assertInCode(t, "type "+k+" struct {", res)
					assertInCode(t, "type "+k+"TagsTuple0 struct {", res)
					// this would fail if it accepts additionalItems because it would come out as []interface{}
					assertInCode(t, "Tags *"+k+"TagsTuple0 `json:\"tags,omitempty\"`", res)
					assertInCode(t, "P0 string `json:\"-\"`", res)
					assertInCode(t, k+"TagsTuple0Items []interface{} `json:\"-\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithItemsAndAdditional2(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithItemsAndAdditional2"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Nil(t, genModel.Items)
			assert.True(t, genModel.IsComplexObject)
			prop := getDefinitionProperty(genModel, "tags")
			assert.True(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				b, err := formatGoFile("with_complex_items.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(b)
					assertInCode(t, "type "+k+" struct {", res)
					assertInCode(t, "type "+k+"TagsTuple0 struct {", res)
					// this would fail if it accepts additionalItems because it would come out as []interface{}
					assertInCode(t, "P0 string `json:\"-\"`", res)
					assertInCode(t, "Tags *"+k+"TagsTuple0 `json:\"tags,omitempty\"`", res)
					assertInCode(t, k+"TagsTuple0Items []*int32 `json:\"-\"`", res)

				}
			}
		}
	}
}

func TestGenerateModel_WithComplexAdditional(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithComplexAdditional"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Nil(t, genModel.Items)
			assert.True(t, genModel.IsComplexObject)
			prop := getDefinitionProperty(genModel, "tags")
			assert.True(t, prop.IsComplexObject)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				b, err := formatGoFile("with_complex_additional.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(b)
					assertInCode(t, "type WithComplexAdditional struct {", res)
					assertInCode(t, "type WithComplexAdditionalTagsTuple0 struct {", res)
					assertInCode(t, "Tags *WithComplexAdditionalTagsTuple0 `json:\"tags,omitempty\"`", res)
					assertInCode(t, "P0 string `json:\"-\"`", res)
					assertInCode(t, "WithComplexAdditionalTagsTuple0Items []*WithComplexAdditionalTagsItems `json:\"-\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_SimpleTuple(t *testing.T) {
	tt := templateTest{t, modelTemplate}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SimpleTuple"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.Empty(t, genModel.ExtraSchemas) {
			assert.True(t, genModel.IsTuple)
			assert.False(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsArray)
			assert.False(t, genModel.IsAnonymous)
			assert.Equal(t, k, genModel.Name)
			assert.Equal(t, k, genModel.GoType)
			assert.Len(t, genModel.Properties, 5)
			buf := bytes.NewBuffer(nil)
			tt.template.Execute(buf, genModel)
			res := buf.String()
			assertInCode(t, "swagger:model "+k, res)
			assertInCode(t, "type "+k+" struct {", res)
			assertInCode(t, "P0 int64 `json:\"-\"`", res)
			assertInCode(t, "P1 string `json:\"-\"`", res)
			assertInCode(t, "P2 strfmt.DateTime `json:\"-\"`", res)
			assertInCode(t, "P3 *Notable `json:\"-\"`", res)
			assertInCode(t, "P4 *Notable `json:\"-\"`", res)
			assertInCode(t, k+") UnmarshalJSON", res)
			assertInCode(t, k+") MarshalJSON", res)
			assertInCode(t, "json.Marshal(data)", res)
			assert.NotRegexp(t, regexp.MustCompile("lastIndex"), res)

			for i, p := range genModel.Properties {
				r := "m.P" + strconv.Itoa(i)
				if !p.IsNullable {
					r = "&" + r
				}
				assertInCode(t, "json.Unmarshal(stage1["+strconv.Itoa(i)+"], "+r+")", res)
				assertInCode(t, "P"+strconv.Itoa(i)+",", res)
			}
		}
	}
}

func TestGenerateModel_TupleWithExtra(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "TupleWithExtra"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.Empty(t, genModel.ExtraSchemas) {
			assert.True(t, genModel.IsTuple)
			assert.False(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsArray)
			assert.False(t, genModel.IsAnonymous)
			assert.True(t, genModel.HasAdditionalItems)
			assert.NotNil(t, genModel.AdditionalItems)
			assert.Equal(t, k, genModel.Name)
			assert.Equal(t, k, genModel.GoType)
			assert.Len(t, genModel.Properties, 4)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("tuple_with_extra.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "swagger:model "+k, res)
					assertInCode(t, "type "+k+" struct {", res)
					assertInCode(t, "P0 int64 `json:\"-\"`", res)
					assertInCode(t, "P1 string `json:\"-\"`", res)
					assertInCode(t, "P2 strfmt.DateTime `json:\"-\"`", res)
					assertInCode(t, "P3 *Notable `json:\"-\"`", res)
					assertInCode(t, k+"Items []*float64 `json:\"-\"`", res)
					assertInCode(t, k+") UnmarshalJSON", res)
					assertInCode(t, k+") MarshalJSON", res)

					for i, p := range genModel.Properties {
						r := "m.P" + strconv.Itoa(i)
						if !p.IsNullable {
							r = "&" + r
						}
						assertInCode(t, "lastIndex = "+strconv.Itoa(i), res)
						assertInCode(t, "json.Unmarshal(stage1["+strconv.Itoa(i)+"], "+r+")", res)
						assertInCode(t, "P"+strconv.Itoa(i)+",", res)
					}
					assertInCode(t, "var lastIndex int", res)
					assertInCode(t, "var toadd *float64", res)
					assertInCode(t, "for _, val := range stage1[lastIndex+1:]", res)
					assertInCode(t, "json.Unmarshal(val, toadd)", res)
					assertInCode(t, "json.Marshal(data)", res)
					assertInCode(t, "for _, v := range m."+k+"Items", res)
				}
			}
		}
	}
}

func TestGenerateModel_TupleWithComplex(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "TupleWithComplex"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) { //&& assert.Empty(t, genModel.ExtraSchemas) {
			assert.True(t, genModel.IsTuple)
			assert.False(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsArray)
			assert.False(t, genModel.IsAnonymous)
			assert.True(t, genModel.HasAdditionalItems)
			assert.NotNil(t, genModel.AdditionalItems)
			assert.Equal(t, k, genModel.Name)
			assert.Equal(t, k, genModel.GoType)
			assert.Len(t, genModel.Properties, 4)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("tuple_with_extra.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "swagger:model "+k, res)
					assertInCode(t, "type "+k+" struct {", res)
					assertInCode(t, "P0 int64 `json:\"-\"`", res)
					assertInCode(t, "P1 string `json:\"-\"`", res)
					assertInCode(t, "P2 strfmt.DateTime `json:\"-\"`", res)
					assertInCode(t, "P3 *Notable `json:\"-\"`", res)
					assertInCode(t, k+"Items []*TupleWithComplexItems `json:\"-\"`", res)
					assertInCode(t, k+") UnmarshalJSON", res)
					assertInCode(t, k+") MarshalJSON", res)

					for i, p := range genModel.Properties {
						r := "m.P" + strconv.Itoa(i)
						if !p.IsNullable {
							r = "&" + r
						}
						assertInCode(t, "lastIndex = "+strconv.Itoa(i), res)
						assertInCode(t, "json.Unmarshal(stage1["+strconv.Itoa(i)+"], "+r+")", res)
						assertInCode(t, "P"+strconv.Itoa(i)+",", res)
					}
					assertInCode(t, "var lastIndex int", res)
					assertInCode(t, "var toadd *TupleWithComplexItems", res)
					assertInCode(t, "for _, val := range stage1[lastIndex+1:]", res)
					assertInCode(t, "json.Unmarshal(val, toadd)", res)
					assertInCode(t, "json.Marshal(data)", res)
					assertInCode(t, "for _, v := range m."+k+"Items", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithTuple(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithTuple"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.NotEmpty(t, genModel.ExtraSchemas) && assert.NotEmpty(t, genModel.Properties) {
			assert.False(t, genModel.IsTuple)
			assert.True(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsArray)
			assert.False(t, genModel.IsAnonymous)

			sch := genModel.ExtraSchemas[0]
			assert.True(t, sch.IsTuple)
			assert.False(t, sch.IsComplexObject)
			assert.False(t, sch.IsArray)
			assert.False(t, sch.IsAnonymous)
			assert.Equal(t, k+"FlagsTuple0", sch.Name)
			assert.False(t, sch.HasAdditionalItems)
			assert.Nil(t, sch.AdditionalItems)

			prop := genModel.Properties[0]
			assert.False(t, genModel.IsTuple)
			assert.True(t, genModel.IsComplexObject)
			assert.False(t, prop.IsArray)
			assert.False(t, prop.IsAnonymous)
			assert.Equal(t, k+"FlagsTuple0", prop.GoType)
			assert.Equal(t, "flags", prop.Name)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("with_tuple.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "swagger:model "+k+"Flags", res)
					assertInCode(t, "type "+k+"FlagsTuple0 struct {", res)
					assertInCode(t, "P0 int64 `json:\"-\"`", res)
					assertInCode(t, "P1 string `json:\"-\"`", res)
					assertInCode(t, k+"FlagsTuple0) UnmarshalJSON", res)
					assertInCode(t, k+"FlagsTuple0) MarshalJSON", res)
					assertInCode(t, "json.Marshal(data)", res)
					assert.NotRegexp(t, regexp.MustCompile("lastIndex"), res)

					for i, p := range sch.Properties {
						r := "m.P" + strconv.Itoa(i)
						if !p.IsNullable {
							r = "&" + r
						}
						assertInCode(t, "json.Unmarshal(stage1["+strconv.Itoa(i)+"], "+r+")", res)
						assertInCode(t, "P"+strconv.Itoa(i)+",", res)
					}
				}
			}
		}
	}
}

func TestGenerateModel_WithTupleWithExtra(t *testing.T) {
	tt := templateTest{t, modelTemplate}
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "WithTupleWithExtra"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) && assert.NotEmpty(t, genModel.ExtraSchemas) && assert.NotEmpty(t, genModel.Properties) {
			assert.False(t, genModel.IsTuple)
			assert.True(t, genModel.IsComplexObject)
			assert.False(t, genModel.IsArray)
			assert.False(t, genModel.IsAnonymous)

			sch := genModel.ExtraSchemas[0]
			assert.True(t, sch.IsTuple)
			assert.False(t, sch.IsComplexObject)
			assert.False(t, sch.IsArray)
			assert.False(t, sch.IsAnonymous)
			assert.Equal(t, k+"FlagsTuple0", sch.Name)
			assert.True(t, sch.HasAdditionalItems)
			assert.NotEmpty(t, sch.AdditionalItems)

			prop := genModel.Properties[0]
			assert.False(t, genModel.IsTuple)
			assert.True(t, genModel.IsComplexObject)
			assert.False(t, prop.IsArray)
			assert.False(t, prop.IsAnonymous)
			assert.Equal(t, k+"FlagsTuple0", prop.GoType)
			assert.Equal(t, "flags", prop.Name)
			buf := bytes.NewBuffer(nil)
			err := tt.template.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("with_tuple.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "swagger:model "+k+"Flags", res)
					assertInCode(t, "type "+k+"FlagsTuple0 struct {", res)
					assertInCode(t, "P0 int64 `json:\"-\"`", res)
					assertInCode(t, "P1 string `json:\"-\"`", res)
					assertInCode(t, k+"FlagsTuple0Items []*float32 `json:\"-\"`", res)
					assertInCode(t, k+"FlagsTuple0) UnmarshalJSON", res)
					assertInCode(t, k+"FlagsTuple0) MarshalJSON", res)
					assertInCode(t, "json.Marshal(data)", res)

					for i, p := range sch.Properties {
						r := "m.P" + strconv.Itoa(i)
						if !p.IsNullable {
							r = "&" + r
						}
						assertInCode(t, "lastIndex = "+strconv.Itoa(i), res)
						assertInCode(t, "json.Unmarshal(stage1["+strconv.Itoa(i)+"], "+r+")", res)
						assertInCode(t, "P"+strconv.Itoa(i)+",", res)
					}

					assertInCode(t, "var lastIndex int", res)
					assertInCode(t, "var toadd *float32", res)
					assertInCode(t, "for _, val := range stage1[lastIndex+1:]", res)
					assertInCode(t, "json.Unmarshal(val, toadd)", res)
					assertInCode(t, "json.Marshal(data)", res)
					assertInCode(t, "for _, v := range m."+k+"FlagsTuple0Items", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithAllOfAndDiscriminator(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["Cat"]
		genModel, err := makeGenDefinition("Cat", "models", schema, specDoc)
		if assert.NoError(t, err) && assert.Len(t, genModel.AllOf, 2) {
			assert.True(t, genModel.IsComplexObject)
			assert.Equal(t, "Cat", genModel.Name)
			assert.Equal(t, "Cat", genModel.GoType)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("cat.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					assertInCode(t, "type Cat struct {", res)
					assertInCode(t, "Pet", res)
					assertInCode(t, "HuntingSkill string `json:\"huntingSkill,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenerateModel_WithAllOf(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["WithAllOf"]
		genModel, err := makeGenDefinition("WithAllOf", "models", schema, specDoc)
		if assert.NoError(t, err) {
			assert.Len(t, genModel.AllOf, 7)
			assert.True(t, genModel.AllOf[1].HasAdditionalProperties)
			assert.True(t, genModel.IsComplexObject)
			assert.Equal(t, "WithAllOf", genModel.Name)
			assert.Equal(t, "WithAllOf", genModel.GoType)
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("all_of_schema.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					//fmt.Println(res)
					assertInCode(t, "type WithAllOf struct {", res)
					assertInCode(t, "type WithAllOfAO2P2 struct {", res)
					assertInCode(t, "type WithAllOfAO3P3 struct {", res)
					assertInCode(t, "type WithAllOfParamsAnon struct {", res)
					assertInCode(t, "type WithAllOfAO4Tuple4 struct {", res)
					assertInCode(t, "type WithAllOfAO5Tuple5 struct {", res)
					assertInCode(t, "Notable", res)
					assertInCode(t, "Title string `json:\"title,omitempty\"`", res)
					assertInCode(t, "Body string `json:\"body,omitempty\"`", res)
					assertInCode(t, "Name *string `json:\"name,omitempty\"`", res)
					assertInCode(t, "P0 float32 `json:\"-\"`", res)
					assertInCode(t, "P0 float64 `json:\"-\"`", res)
					assertInCode(t, "P1 strfmt.DateTime `json:\"-\"`", res)
					assertInCode(t, "P1 strfmt.Date `json:\"-\"`", res)
					assertInCode(t, "Opinion *string `json:\"opinion,omitempty\"`", res)
					assertInCode(t, "WithAllOfAO5Tuple5Items []*strfmt.Password `json:\"-\"`", res)
					assertInCode(t, "AO1 map[string]int32 `json:\"-\"`", res)
					assertInCode(t, "WithAllOfAO2P2 map[string]int64 `json:\"-\"`", res)
				}
			}
		}
	}
}

func findProperty(properties []GenSchema, name string) *GenSchema {
	for _, p := range properties {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func getDefinitionProperty(genModel *GenDefinition, name string) *GenSchema {
	return findProperty(genModel.Properties, name)
}

func TestNumericKeys(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/bugs/162/swagger.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["AvatarUrls"]
		genModel, err := makeGenDefinition("AvatarUrls", "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("all_of_schema.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					assertInCode(t, "Nr16x16 *string `json:\"16x16,omitempty\"`", res)
				}
			}
		}
	}
}

func TestGenModel_Issue196(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/bugs/196/swagger.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		schema := definitions["Event"]
		genModel, err := makeGenDefinition("Event", "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("primitive_event.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					assertInCode(t, "Event) Validate(formats strfmt.Registry) error", res)
				}
			}
		}
	}
}

func TestGenModel_Issue222(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/tasklist.basic.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "Price"
		genModel, err := makeGenDefinition(k, "models", definitions[k], specDoc)
		if assert.NoError(t, err) && assert.True(t, genModel.HasValidations) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("price.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					assertInCode(t, "Price) Validate(formats strfmt.Registry) error", res)
					assertInCode(t, "Currency *Currency `json:\"currency,omitempty\"`", res)
					assertInCode(t, "m.Currency.Validate(formats); err != nil", res)
				}
			}
		}
	}
}

func TestGenModel_Issue243(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "HasDynMeta"
		genModel, err := makeGenDefinition(k, "models", definitions[k], specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("has_dyn_meta.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					if !assertInCode(t, "Metadata DynamicMetaData `json:\"metadata,omitempty\"`", res) {
						fmt.Println(res)
					}
				}
			}
		}
	}
}

func TestGenModel_Issue252(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/bugs/252/swagger.json")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SodaBrand"
		genModel, err := makeGenDefinition(k, "models", definitions[k], specDoc)
		if assert.NoError(t, err) && assert.False(t, genModel.IsNullable) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("soda_brand.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)
					b1 := assertInCode(t, "type "+k+" string", res)
					b2 := assertInCode(t, "(m "+k+") validateSodaBrand", res)
					b3 := assertInCode(t, "(m "+k+") Validate", res)
					if !(b1 && b2 && b3) {
						fmt.Println(res)
					}
				}
			}
		}
	}
}

func TestGenModel_Issue251(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/bugs/251/swagger.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "example"
		genModel, err := makeGenDefinition(k, "models", definitions[k], specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("example.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)

					b1 := assertInCode(t, "type "+swag.ToGoName(k)+" struct", res)
					b2 := assertInCode(t, "Begin strfmt.DateTime `json:\"begin,omitempty\"`", res)
					b3 := assertInCode(t, "End *strfmt.DateTime `json:\"end,omitempty\"`", res)
					b4 := assertInCode(t, "Name *string `json:\"name,omitempty\"`", res)
					b5 := assertInCode(t, "(m *"+swag.ToGoName(k)+") validateBegin", res)
					//b6 := assertInCode(t, "(m *"+swag.ToGoName(k)+") validateEnd", res)
					b7 := assertInCode(t, "(m *"+swag.ToGoName(k)+") Validate", res)
					if !(b1 && b2 && b3 && b4 && b5 && b7) {
						fmt.Println(res)
					}
				}
			}
		}
	}
}

func TestGenModel_Issue257(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.models.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "HasSpecialCharProp"
		genModel, err := makeGenDefinition(k, "models", definitions[k], specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ct, err := formatGoFile("example.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ct)

					b1 := assertInCode(t, "type "+swag.ToGoName(k)+" struct", res)
					b2 := assertInCode(t, "AtType *string `json:\"@type,omitempty\"`", res)
					b3 := assertInCode(t, "Type *string `json:\"type,omitempty\"`", res)
					if !(b1 && b2 && b3) {
						fmt.Println(res)
					}
				}
			}
		}
	}
}
