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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/stretchr/testify/assert"
)

func TestUniqueOperationNames(t *testing.T) {
	doc, err := spec.Load("../fixtures/codegen/todolist.simple.yml")
	if assert.NoError(t, err) {
		sp := doc.Spec()
		sp.Paths.Paths["/tasks"].Post.ID = "saveTask"
		sp.Paths.Paths["/tasks"].Post.AddExtension("origName", "createTask")
		sp.Paths.Paths["/tasks/{id}"].Put.ID = "saveTask"
		sp.Paths.Paths["/tasks/{id}"].Put.AddExtension("origName", "updateTask")

		ops := gatherOperations(doc, nil)
		assert.Len(t, ops, 4)
		_, exists := ops["saveTask"]
		assert.True(t, exists)
		_, exists = ops["PutTasksID"]
		assert.True(t, exists)
	}
}

func TestEmptyOperationNames(t *testing.T) {
	doc, err := spec.Load("../fixtures/codegen/todolist.simple.yml")
	if assert.NoError(t, err) {
		sp := doc.Spec()
		sp.Paths.Paths["/tasks"].Post.ID = ""
		sp.Paths.Paths["/tasks"].Post.AddExtension("origName", "createTask")
		sp.Paths.Paths["/tasks/{id}"].Put.ID = ""
		sp.Paths.Paths["/tasks/{id}"].Put.AddExtension("origName", "updateTask")

		ops := gatherOperations(doc, nil)
		assert.Len(t, ops, 4)
		_, exists := ops["PostTasks"]
		assert.True(t, exists)
		_, exists = ops["PutTasksID"]
		assert.True(t, exists)
	}
}

func TestMakeResponseHeader(t *testing.T) {
	b, err := opBuilder("getTasks", "")
	if assert.NoError(t, err) {
		hdr := findResponseHeader(&b.Operation, 200, "X-Rate-Limit")
		gh := b.MakeHeader("a", "X-Rate-Limit", *hdr)
		assert.True(t, gh.IsPrimitive)
		assert.Equal(t, "int32", gh.GoType)
		assert.Equal(t, "X-Rate-Limit", gh.Name)
	}
}

func TestMakeResponseHeaderDefaultValues(t *testing.T) {
	b, err := opBuilder("getTasks", "")
	if assert.NoError(t, err) {
		var testCases = []struct {
			name         string      // input
			typeStr      string      // expected type
			defaultValue interface{} // expected result
		}{
			{"Access-Control-Allow-Origin", "string", "*"},
			{"X-Rate-Limit", "int32", nil},
			{"X-Rate-Limit-Remaining", "int32", float64(42)},
			{"X-Rate-Limit-Reset", "int32", "1449875311"},
			{"X-Rate-Limit-Reset-Human", "string", "3 days"},
			{"X-Rate-Limit-Reset-Human-Number", "string", float64(3)},
		}

		for _, tc := range testCases {
			// t.Logf("tc: %+v", tc)
			hdr := findResponseHeader(&b.Operation, 200, tc.name)
			assert.NotNil(t, hdr)
			gh := b.MakeHeader("a", tc.name, *hdr)
			assert.True(t, gh.IsPrimitive)
			assert.Equal(t, tc.typeStr, gh.GoType)
			assert.Equal(t, tc.name, gh.Name)
			assert.Exactly(t, tc.defaultValue, gh.Default)
		}
	}
}

func TestMakeResponse(t *testing.T) {
	b, err := opBuilder("getTasks", "")
	if assert.NoError(t, err) {
		resolver := &typeResolver{ModelsPackage: b.ModelsPackage, Doc: b.Doc}
		resolver.KnownDefs = make(map[string]struct{})
		for k := range b.Doc.Spec().Definitions {
			resolver.KnownDefs[k] = struct{}{}
		}
		gO, err := b.MakeResponse("a", "getTasksSuccess", true, resolver, 200, b.Operation.Responses.StatusCodeResponses[200])
		if assert.NoError(t, err) {
			assert.Len(t, gO.Headers, 6)
			assert.NotNil(t, gO.Schema)
			assert.True(t, gO.Schema.IsArray)
			assert.NotNil(t, gO.Schema.Items)
			assert.False(t, gO.Schema.IsAnonymous)
			assert.Equal(t, "[]*models.Task", gO.Schema.GoType)
		}
	}
}

func TestMakeResponse_WithAllOfSchema(t *testing.T) {
	b, err := methodPathOpBuilder("get", "/media/search", "../fixtures/codegen/instagram.yml")
	if assert.NoError(t, err) {
		resolver := &typeResolver{ModelsPackage: b.ModelsPackage, Doc: b.Doc}
		resolver.KnownDefs = make(map[string]struct{})
		for k := range b.Doc.Spec().Definitions {
			resolver.KnownDefs[k] = struct{}{}
		}
		gO, err := b.MakeResponse("a", "get /media/search", true, resolver, 200, b.Operation.Responses.StatusCodeResponses[200])
		if assert.NoError(t, err) {
			if assert.NotNil(t, gO.Schema) {
				assert.Equal(t, "GetMediaSearchBodyBody", gO.Schema.GoType)
			}
			if assert.NotEmpty(t, b.ExtraSchemas) {
				body := b.ExtraSchemas["GetMediaSearchBodyBody"]
				if assert.NotEmpty(t, body.Properties) {
					prop := body.Properties[0]
					assert.Equal(t, "data", prop.Name)
					assert.Equal(t, "[]*DataItems0", prop.GoType)
				}
				items := b.ExtraSchemas["DataItems0"]
				if assert.NotEmpty(t, items.AllOf) {
					media := items.AllOf[0]
					assert.Equal(t, "models.Media", media.GoType)
				}
			}
		}
	}
}

func TestMakeOperationParam(t *testing.T) {
	b, err := opBuilder("getTasks", "")
	if assert.NoError(t, err) {
		resolver := &typeResolver{ModelsPackage: b.ModelsPackage, Doc: b.Doc}
		gO, err := b.MakeParameter("a", resolver, b.Operation.Parameters[0])
		if assert.NoError(t, err) {
			assert.Equal(t, "size", gO.Name)
			assert.True(t, gO.IsPrimitive)
		}
	}
}

func TestMakeOperationParamItem(t *testing.T) {
	b, err := opBuilder("arrayQueryParams", "../fixtures/codegen/todolist.arrayquery.yml")
	if assert.NoError(t, err) {
		resolver := &typeResolver{ModelsPackage: b.ModelsPackage, Doc: b.Doc}
		gO, err := b.MakeParameterItem("a", "siString", "i", "siString", "a.SiString", "query", resolver, b.Operation.Parameters[1].Items, nil)
		if assert.NoError(t, err) {
			assert.Nil(t, gO.Parent)
			assert.True(t, gO.IsPrimitive)
		}
	}
}

func TestMakeOperation(t *testing.T) {
	b, err := opBuilder("getTasks", "")
	if assert.NoError(t, err) {
		gO, err := b.MakeOperation()
		if assert.NoError(t, err) {
			//pretty.Println(gO)
			assert.Equal(t, "getTasks", gO.Name)
			assert.Equal(t, "GET", gO.Method)
			assert.Equal(t, "/tasks", gO.Path)
			assert.Len(t, gO.Params, 2)
			assert.Len(t, gO.Responses, 1)
			assert.NotNil(t, gO.DefaultResponse)
			assert.NotNil(t, gO.SuccessResponse)
		}

		// TODO: validate rendering of a complex operation
	}
}

func TestRenderOperation_InstagramSearch(t *testing.T) {
	b, err := methodPathOpBuilder("get", "/media/search", "../fixtures/codegen/instagram.yml")
	if assert.NoError(t, err) {
		gO, err := b.MakeOperation()
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := operationTemplate.Execute(buf, gO)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("operation.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					//fmt.Println(res)
					assertInCode(t, "Data []*DataItems0 `json:\"data,omitempty\"`", res)
					assertInCode(t, "models.Media", res)
				}
			}
		}
	}
}

func methodPathOpBuilder(method, path, fname string) (codeGenOpBuilder, error) {
	if fname == "" {
		fname = "../fixtures/codegen/todolist.simple.yml"
	}

	specDoc, err := spec.Load(fname)
	if err != nil {
		return codeGenOpBuilder{}, err
	}

	op, ok := specDoc.OperationFor(method, path)
	if !ok {
		return codeGenOpBuilder{}, errors.New("No operation could be found for " + method + " " + path)
	}

	return codeGenOpBuilder{
		Name:          method + " " + path,
		Method:        method,
		Path:          path,
		APIPackage:    "restapi",
		ModelsPackage: "models",
		Principal:     "models.User",
		Target:        ".",
		Operation:     *op,
		Doc:           specDoc,
		Authed:        false,
		ExtraSchemas:  make(map[string]GenSchema),
	}, nil
}

func opBuilder(name, fname string) (codeGenOpBuilder, error) {
	if fname == "" {
		fname = "../fixtures/codegen/todolist.simple.yml"
	}

	specDoc, err := spec.Load(fname)
	if err != nil {
		return codeGenOpBuilder{}, err
	}

	method, path, op, ok := specDoc.OperationForName(name)
	if !ok {
		return codeGenOpBuilder{}, errors.New("No operation could be found for " + name)
	}

	return codeGenOpBuilder{
		Name:          name,
		Method:        method,
		Path:          path,
		APIPackage:    "restapi",
		ModelsPackage: "models",
		Principal:     "models.User",
		Target:        ".",
		Operation:     *op,
		Doc:           specDoc,
		Authed:        false,
		ExtraSchemas:  make(map[string]GenSchema),
	}, nil
}

func findResponseHeader(op *spec.Operation, code int, name string) *spec.Header {
	resp := op.Responses.Default
	if code > 0 {
		bb, ok := op.Responses.StatusCodeResponses[code]
		if ok {
			resp = &bb
		}
	}

	if resp == nil {
		return nil
	}

	hdr, ok := resp.Headers[name]
	if !ok {
		return nil
	}

	return &hdr
}

func TestDateFormat_Spec1(t *testing.T) {
	b, err := opBuilder("putTesting", "../fixtures/bugs/193/spec1.json")
	if assert.NoError(t, err) {
		op, err := b.MakeOperation()
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := clientParamTemplate.Execute(buf, op)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("put_testing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "frTestingThis.String()", res)
				} else {
					fmt.Println(buf.String())
				}
			}
		}
	}
}

func TestDateFormat_Spec2(t *testing.T) {
	b, err := opBuilder("putTesting", "../fixtures/bugs/193/spec2.json")
	if assert.NoError(t, err) {
		op, err := b.MakeOperation()
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := clientParamTemplate.Execute(buf, op)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("put_testing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "valuesTestingThis = append(valuesTestingThis, v.String())", res)
				} else {
					fmt.Println(buf.String())
				}
			}
		}
	}
}

func TestBuilder_Issue287(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)
	dr, _ := os.Getwd()
	appGen, err := newAppGenerator("plainTexter", nil, nil, &GenOpts{
		Spec:              filepath.FromSlash("../fixtures/bugs/287/swagger.yml"),
		IncludeModel:      true,
		IncludeValidator:  true,
		IncludeHandler:    true,
		IncludeParameters: true,
		IncludeResponses:  true,
		IncludeMain:       true,
		APIPackage:        "restapi",
		ModelPackage:      "model",
		ServerPackage:     "server",
		ClientPackage:     "client",
		Target:            dr,
	})
	if assert.NoError(t, err) {
		op, err := appGen.makeCodegenApp()
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := builderTemplate.Execute(buf, op)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("put_testing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "case \"text/plain\":", res)
				} else {
					fmt.Println(buf.String())
				}
			}
		}
	}
}
