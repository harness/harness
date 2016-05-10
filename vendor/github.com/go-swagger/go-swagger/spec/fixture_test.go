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
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-swagger/go-swagger/swag"
	. "github.com/smartystreets/goconvey/convey"
)

var extensions = []string{"json"}

func roundTripTest(t *testing.T, fixtureType, extension, fileName string, schema interface{}) {
	if extension == "yaml" {
		roundTripTestYAML(t, fixtureType, fileName, schema)
	} else {
		roundTripTestJSON(t, fixtureType, fileName, schema)
	}

}

func roundTripTestJSON(t *testing.T, fixtureType, fileName string, schema interface{}) {
	specName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	Convey("verifying "+fixtureType+" JSON fixture "+specName, t, func() {
		b, err := ioutil.ReadFile(fileName)
		So(err, ShouldBeNil)
		Println()
		var expected map[string]interface{}
		err = json.Unmarshal(b, &expected)
		So(err, ShouldBeNil)

		err = json.Unmarshal(b, schema)
		So(err, ShouldBeNil)

		cb, err := json.MarshalIndent(schema, "", "  ")
		So(err, ShouldBeNil)

		var actual map[string]interface{}
		err = json.Unmarshal(cb, &actual)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEquivalentTo, expected)
	})
}

func roundTripTestYAML(t *testing.T, fixtureType, fileName string, schema interface{}) {
	specName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	Convey("verifying "+fixtureType+" YAML fixture "+specName, t, func() {
		b, err := swag.YAMLDoc(fileName)
		So(err, ShouldBeNil)
		Println()
		var expected map[string]interface{}
		err = json.Unmarshal(b, &expected)
		So(err, ShouldBeNil)

		err = json.Unmarshal(b, schema)
		So(err, ShouldBeNil)

		cb, err := json.MarshalIndent(schema, "", "  ")
		So(err, ShouldBeNil)

		var actual map[string]interface{}
		err = json.Unmarshal(cb, &actual)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEquivalentTo, expected)
	})
}

func TestPropertyFixtures(t *testing.T) {
	for _, extension := range extensions {
		path := filepath.Join("..", "fixtures", extension, "models", "properties")
		files, err := ioutil.ReadDir(path)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			roundTripTest(t, "property", extension, filepath.Join(path, f.Name()), &Schema{})
		}
	}
}

func TestAdditionalPropertiesWithObject(t *testing.T) {
	schema := new(Schema)
	Convey("verifying model with map with object value", t, func() {
		b, err := swag.YAMLDoc("../fixtures/yaml/models/modelWithObjectMap.yaml")
		So(err, ShouldBeNil)
		Println()

		var expected map[string]interface{}
		err = json.Unmarshal(b, &expected)
		So(err, ShouldBeNil)

		err = json.Unmarshal(b, schema)
		So(err, ShouldBeNil)

		cb, err := json.MarshalIndent(schema, "", "  ")
		So(err, ShouldBeNil)

		var actual map[string]interface{}
		err = json.Unmarshal(cb, &actual)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEquivalentTo, expected)
	})
}

func TestModelFixtures(t *testing.T) {
	path := filepath.Join("..", "fixtures", "json", "models")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	specs := []string{"modelWithObjectMap", "models", "modelWithComposition", "modelWithExamples", "multipleModels"}
FILES:
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		for _, spec := range specs {
			if strings.HasPrefix(f.Name(), spec) {
				roundTripTest(t, "model", "json", filepath.Join(path, f.Name()), &Schema{})
				continue FILES
			}
		}
		//fmt.Println("trying", f.Name())
		roundTripTest(t, "model", "json", filepath.Join(path, f.Name()), &Schema{})
	}
	path = filepath.Join("..", "fixtures", "yaml", "models")
	files, err = ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
YAMLFILES:
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		for _, spec := range specs {
			if strings.HasPrefix(f.Name(), spec) {
				roundTripTest(t, "model", "yaml", filepath.Join(path, f.Name()), &Schema{})
				continue YAMLFILES
			}
		}
		// fmt.Println("trying", f.Name())
		roundTripTest(t, "model", "yaml", filepath.Join(path, f.Name()), &Schema{})
	}
}

func TestParameterFixtures(t *testing.T) {
	path := filepath.Join("..", "fixtures", "json", "resources", "parameters")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		roundTripTest(t, "parameter", "json", filepath.Join(path, f.Name()), &Parameter{})
	}
}

func TestOperationFixtures(t *testing.T) {
	path := filepath.Join("..", "fixtures", "json", "resources", "operations")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		roundTripTest(t, "operation", "json", filepath.Join(path, f.Name()), &Operation{})
	}
}

func TestResponseFixtures(t *testing.T) {
	path := filepath.Join("..", "fixtures", "json", "responses")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Name(), "multiple") {
			roundTripTest(t, "response", "json", filepath.Join(path, f.Name()), &Response{})
		} else {
			roundTripTest(t, "responses", "json", filepath.Join(path, f.Name()), &Responses{})
		}
	}
}

func TestResourcesFixtures(t *testing.T) {
	path := filepath.Join("..", "fixtures", "json", "resources")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	pathItems := []string{"resourceWithLinkedDefinitions_part1"}
	toSkip := []string{}
FILES:
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		for _, ts := range toSkip {
			if strings.HasPrefix(f.Name(), ts) {
				Convey("verifying resource"+strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())), t, func() {
					b, err := ioutil.ReadFile(filepath.Join(path, f.Name()))
					So(err, ShouldBeNil)
					verifySpecJSON(b)
				})
				continue FILES
			}
		}
		for _, pi := range pathItems {
			if strings.HasPrefix(f.Name(), pi) {
				roundTripTest(t, "path items", "json", filepath.Join(path, f.Name()), &PathItem{})
				continue FILES
			}
		}
		Convey("verifying resource "+strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())), t, func() {
			b, err := ioutil.ReadFile(filepath.Join(path, f.Name()))
			So(err, ShouldBeNil)
			verifySpecJSON(b)
		})

	}
}
