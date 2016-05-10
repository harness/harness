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
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/strfmt"
	"github.com/go-swagger/go-swagger/swag"
	. "github.com/smartystreets/goconvey/convey"
)

type schemaTestT struct {
	Description string       `json:"description"`
	Schema      *spec.Schema `json:"schema"`
	Tests       []struct {
		Description string      `json:"description"`
		Data        interface{} `json:"data"`
		Valid       bool        `json:"valid"`
	}
}

var jsonSchemaFixturesPath = filepath.Join("..", "..", "fixtures", "jsonschema_suite")

var ints = []interface{}{
	1,
	int8(1),
	int16(1),
	int(1),
	int32(1),
	int64(1),
	uint8(1),
	uint16(1),
	uint(1),
	uint32(1),
	uint64(1),
	5.0,
	float32(5.0),
	float64(5.0),
}

var notInts = []interface{}{
	5.1,
	float32(5.1),
	float64(5.1),
}

var notNumbers = []interface{}{
	map[string]string{},
	struct{}{},
	time.Time{},
	"yada",
}

var enabled = []string{
	"minLength",
	"maxLength",
	"pattern",
	"type",
	"minimum",
	"maximum",
	"multipleOf",
	"enum",
	"default",
	"dependencies",
	"items",
	"maxItems",
	"maxProperties",
	"minItems",
	"minProperties",
	"patternProperties",
	"required",
	"additionalItems",
	"uniqueItems",
	"properties",
	"additionalProperties",
	"allOf",
	"not",
	"oneOf",
	"anyOf",
	"ref",
	"definitions",
	"refRemote",
	"format",
}

type noopResCache struct {
}

func (n *noopResCache) Get(key string) (interface{}, bool) {
	return nil, false
}
func (n *noopResCache) Set(string, interface{}) {

}

func isEnabled(nm string) bool {
	return swag.ContainsStringsCI(enabled, nm)
}

func TestJSONSchemaSuite(t *testing.T) {
	go func() {
		err := http.ListenAndServe("localhost:1234", http.FileServer(http.Dir(jsonSchemaFixturesPath+"/remotes")))
		if err != nil {
			panic(err.Error())
		}
	}()
	Convey("The JSON Schema test suite", t, func() {
		files, err := ioutil.ReadDir(jsonSchemaFixturesPath)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			fileName := f.Name()
			specName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			if isEnabled(specName) {

				Convey("for "+specName, func() {
					b, _ := ioutil.ReadFile(filepath.Join(jsonSchemaFixturesPath, fileName))

					var testDescriptions []schemaTestT
					json.Unmarshal(b, &testDescriptions)

					for _, testDescription := range testDescriptions {

						Convey(testDescription.Description, func() {

							So(spec.ExpandSchema(testDescription.Schema, nil, nil /*new(noopResCache)*/), ShouldBeNil)

							validator := NewSchemaValidator(testDescription.Schema, nil, "data", strfmt.Default)

							for _, test := range testDescription.Tests {

								Convey(test.Description, func() {

									result := validator.Validate(test.Data)
									So(result, ShouldNotBeNil)

									if test.Valid {
										So(result.Errors, ShouldBeEmpty)

									} else {
										So(result.Errors, ShouldNotBeEmpty)
									}
								})
							}
						})
					}
				})
			}
		}
	})
}
