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
	gobuild "go/build"
	goparser "go/parser"
	"log"
	"path/filepath"
	"sort"
	"testing"

	"golang.org/x/tools/go/loader"

	"github.com/stretchr/testify/assert"
)

func TestAnnotationMatcher(t *testing.T) {
	variations := []string{
		"// swagger",
		" swagger",
		"swagger",
		" * swagger",
	}
	known := []string{
		"meta",
		"route",
		"model",
		"parameters",
		"strfmt",
		"response",
	}

	for _, variation := range variations {
		for _, tpe := range known {
			assert.True(t, rxSwaggerAnnotation.MatchString(variation+":"+tpe))
		}
	}
}

func classifierProgram() *loader.Program {
	var ldr loader.Config
	ldr.ParserMode = goparser.ParseComments
	ldr.Build = &gobuild.Default
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/classification")
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/classification/models")
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/classification/operations")
	prog, err := ldr.Load()
	if err != nil {
		log.Fatal(err)
	}
	return prog
}

func petstoreProgram() *loader.Program {
	var ldr loader.Config
	ldr.ParserMode = goparser.ParseComments
	ldr.Build = &gobuild.Default
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/petstore")
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/petstore/models")
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/petstore/rest/handlers")
	prog, err := ldr.Load()
	if err != nil {
		log.Fatal(err)
	}
	return prog
}

func invalidProgram(name string) *loader.Program {
	var ldr loader.Config
	ldr.ParserMode = goparser.ParseComments
	ldr.ImportWithTests("github.com/go-swagger/go-swagger/fixtures/goparsing/" + name)
	prog, err := ldr.Load()
	if err != nil {
		log.Fatal(err)
	}
	return prog
}

func testInvalidProgram(t testing.TB, name string) bool {
	prog := invalidProgram(name)
	classifier := &programClassifier{}
	_, err := classifier.Classify(prog)
	return assert.Error(t, err)
}

func TestDuplicateAnnotations(t *testing.T) {
	for _, b := range []string{"model_param", "param_model", "model_response", "response_model"} {
		testInvalidProgram(t, "invalid_"+b)
	}
}

func TestClassifier(t *testing.T) {

	prog := petstoreProgram()
	classifier := &programClassifier{}
	classified, err := classifier.Classify(prog)
	assert.NoError(t, err)

	// ensure all the dependencies are there
	assert.Len(t, classified.Meta, 1)
	assert.Len(t, classified.Operations, 2)

	var fNames []string
	for _, file := range classified.Models {
		fNames = append(
			fNames,
			filepath.Base(prog.Fset.File(file.Pos()).Name()))
	}

	sort.Sort(sort.StringSlice(fNames))
	assert.EqualValues(t, []string{"order.go", "pet.go", "tag.go", "user.go"}, fNames)
}

func TestClassifierInclude(t *testing.T) {

	prog := classificationProg
	classifier := &programClassifier{
		Includes: packageFilters([]packageFilter{
			packageFilter{"github.com/go-swagger/go-swagger/fixtures/goparsing/classification"},
			packageFilter{"github.com/go-swagger/go-swagger/fixtures/goparsing/classification/transitive/mods"},
			packageFilter{"github.com/go-swagger/go-swagger/fixtures/goparsing/classification/operations"},
		}),
	}
	classified, err := classifier.Classify(prog)
	assert.NoError(t, err)

	// ensure all the dependencies are there
	assert.Len(t, classified.Meta, 1)
	assert.Len(t, classified.Operations, 1)

	//var fNames []string
	//for _, file := range classified.Models {
	//fNames = append(
	//fNames,
	//filepath.Base(prog.Fset.File(file.Pos()).Name()))
	//}

	//sort.Sort(sort.StringSlice(fNames))
	//assert.EqualValues(t, []string{"pet.go"}, fNames)
}

func TestClassifierExclude(t *testing.T) {

	prog := classificationProg
	classifier := &programClassifier{
		Excludes: packageFilters([]packageFilter{
			packageFilter{"github.com/go-swagger/go-swagger/fixtures/goparsing/classification/transitive/mods"},
		}),
	}
	classified, err := classifier.Classify(prog)
	assert.NoError(t, err)

	// ensure all the dependencies are there
	assert.Len(t, classified.Meta, 1)
	assert.Len(t, classified.Operations, 1)

	//var fNames []string
	//for _, file := range classified.Models {
	//fNames = append(
	//fNames,
	//filepath.Base(prog.Fset.File(file.Pos()).Name()))
	//}

	//sort.Sort(sort.StringSlice(fNames))
	//assert.EqualValues(t, []string{"order.go", "user.go"}, fNames)
}
