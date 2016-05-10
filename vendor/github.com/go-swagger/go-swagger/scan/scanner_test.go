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
	"fmt"
	"go/ast"
	goparser "go/parser"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/loader"

	_ "github.com/go-swagger/scan-repo-boundary/makeplans"
)

var classificationProg *loader.Program
var noModelDefs map[string]spec.Schema

func init() {
	classificationProg = classifierProgram()
	docFile := "../fixtures/goparsing/classification/models/nomodel.go"
	fileTree, err := goparser.ParseFile(classificationProg.Fset, docFile, nil, goparser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	sp := newSchemaParser(classificationProg)
	noModelDefs = make(map[string]spec.Schema)
	err = sp.Parse(fileTree, noModelDefs)
	if err != nil {
		log.Fatal(err)
	}
}

func TestAppScanner_NewSpec(t *testing.T) {
	scanner, err := newAppScanner(&Opts{BasePath: "../fixtures/goparsing/petstore/petstore-fixture"}, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, scanner)
	doc, err := scanner.Parse()
	assert.NoError(t, err)
	if assert.NotNil(t, doc) {
		verifyParsedPetStore(t, doc)
	}
}

func TestAppScanner_BookingCrossRepo(t *testing.T) {
	scanner, err := newAppScanner(&Opts{BasePath: "../fixtures/goparsing/bookings"}, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, scanner)
	doc, err := scanner.Parse()
	assert.NoError(t, err)
	if assert.NotNil(t, doc) {
		_, ok := doc.Definitions["BookingResponse"]
		assert.False(t, ok)
	}
}

func verifyParsedPetStore(t testing.TB, doc *spec.Swagger) {
	assert.EqualValues(t, []string{"application/json"}, doc.Consumes)
	assert.EqualValues(t, []string{"application/json"}, doc.Produces)
	assert.EqualValues(t, []string{"http", "https"}, doc.Schemes)
	assert.Equal(t, "localhost", doc.Host)
	assert.Equal(t, "/v2", doc.BasePath)

	verifyInfo(t, doc.Info)

	if assert.NotNil(t, doc.Paths) {
		assert.Len(t, doc.Paths.Paths, 4)
	}
	assert.Len(t, doc.Definitions, 3)
	assert.Len(t, doc.Responses, 3)

	definitions := doc.Definitions
	mod, ok := definitions["tag"]
	assert.True(t, ok)
	assert.Equal(t, spec.StringOrArray([]string{"object"}), mod.Type)
	assert.Equal(t, "A Tag is an extra piece of data to provide more information about a pet.", mod.Title)
	assert.Equal(t, "It is used to describe the animals available in the store.", mod.Description)
	assert.Len(t, mod.Required, 2)

	assertProperty(t, &mod, "integer", "id", "int64", "ID")
	prop, ok := mod.Properties["id"]
	assert.True(t, ok, "should have had an 'id' property")
	assert.Equal(t, "The id of the tag.", prop.Description)

	assertProperty(t, &mod, "string", "value", "", "Value")
	prop, ok = mod.Properties["value"]
	assert.Equal(t, "The value of the tag.", prop.Description)

	mod, ok = definitions["pet"]
	assert.True(t, ok)
	assert.Equal(t, spec.StringOrArray([]string{"object"}), mod.Type)
	assert.Equal(t, "A Pet is the main product in the store.", mod.Title)
	assert.Equal(t, "It is used to describe the animals available in the store.", mod.Description)
	assert.Len(t, mod.Required, 2)

	assertProperty(t, &mod, "integer", "id", "int64", "ID")
	prop, ok = mod.Properties["id"]
	assert.True(t, ok, "should have had an 'id' property")
	assert.Equal(t, "The id of the pet.", prop.Description)

	assertProperty(t, &mod, "string", "name", "", "Name")
	prop, ok = mod.Properties["name"]
	assert.Equal(t, "The name of the pet.", prop.Description)
	assert.EqualValues(t, 3, *prop.MinLength)
	assert.EqualValues(t, 50, *prop.MaxLength)
	assert.Equal(t, "\\w[\\w-]+", prop.Pattern)

	assertArrayProperty(t, &mod, "string", "photoUrls", "", "PhotoURLs")
	prop, ok = mod.Properties["photoUrls"]
	assert.Equal(t, "The photo urls for the pet.\nThis only accepts jpeg or png images.", prop.Description)
	if assert.NotNil(t, prop.Items) && assert.NotNil(t, prop.Items.Schema) {
		assert.Equal(t, "\\.(jpe?g|png)$", prop.Items.Schema.Pattern)
	}

	assertProperty(t, &mod, "string", "status", "", "Status")
	prop, ok = mod.Properties["status"]
	assert.Equal(t, "The current status of the pet in the store.", prop.Description)

	assertArrayRef(t, &mod, "tags", "Tags", "#/definitions/tag")
	prop, ok = mod.Properties["tags"]
	assert.True(t, ok)
	assert.Equal(t, "Extra bits of information attached to this pet.", prop.Description)

	mod, ok = definitions["order"]
	assert.True(t, ok)
	assert.Len(t, mod.Properties, 4)
	assert.Len(t, mod.Required, 3)

	assertProperty(t, &mod, "integer", "id", "int64", "ID")
	prop, ok = mod.Properties["id"]
	assert.True(t, ok, "should have had an 'id' property")
	assert.Equal(t, "the ID of the order", prop.Description)

	assertProperty(t, &mod, "integer", "userId", "int64", "UserID")
	prop, ok = mod.Properties["userId"]
	assert.True(t, ok, "should have had an 'userId' property")
	assert.Equal(t, "the id of the user who placed the order.", prop.Description)

	assertProperty(t, &mod, "string", "orderedAt", "date-time", "OrderedAt")
	prop, ok = mod.Properties["orderedAt"]
	assert.Equal(t, "the time at which this order was made.", prop.Description)
	assert.True(t, ok, "should have an 'orderedAt' property")

	assertArrayProperty(t, &mod, "object", "items", "", "Items")
	prop, ok = mod.Properties["items"]
	assert.True(t, ok, "should have an 'items' slice")
	assert.NotNil(t, prop.Items, "items should have had an items property")
	assert.NotNil(t, prop.Items.Schema, "items.items should have had a schema property")

	itprop := prop.Items.Schema
	assert.Len(t, itprop.Properties, 2)
	assert.Len(t, itprop.Required, 2)

	assertProperty(t, itprop, "integer", "petId", "int64", "PetID")
	iprop, ok := itprop.Properties["petId"]
	assert.True(t, ok, "should have had a 'petId' property")
	assert.Equal(t, "the id of the pet to order", iprop.Description)

	assertProperty(t, itprop, "integer", "qty", "int32", "Quantity")
	iprop, ok = itprop.Properties["qty"]
	assert.True(t, ok, "should have had a 'qty' property")
	assert.Equal(t, "the quantity of this pet to order", iprop.Description)
	assert.EqualValues(t, 1, *iprop.Minimum)

	// responses
	resp, ok := doc.Responses["genericError"]
	assert.True(t, ok)
	assert.NotNil(t, resp.Schema)
	assert.Len(t, resp.Schema.Properties, 2)
	assertProperty(t, resp.Schema, "integer", "code", "int32", "Code")
	assertProperty(t, resp.Schema, "string", "message", "", "Message")

	resp, ok = doc.Responses["validationError"]
	assert.True(t, ok)
	assert.NotNil(t, resp.Schema)
	assert.Len(t, resp.Schema.Properties, 3)
	assertProperty(t, resp.Schema, "integer", "code", "int32", "Code")
	assertProperty(t, resp.Schema, "string", "message", "", "Message")
	assertProperty(t, resp.Schema, "string", "field", "", "Field")

	paths := doc.Paths.Paths

	// path /pets
	op, ok := paths["/pets"]
	assert.True(t, ok)
	assert.NotNil(t, op)

	// listPets
	assert.NotNil(t, op.Get)
	assert.Equal(t, "Lists the pets known to the store.", op.Get.Summary)
	assert.Equal(t, "By default it will only lists pets that are available for sale.\nThis can be changed with the status flag.", op.Get.Description)
	assert.Equal(t, "listPets", op.Get.ID)
	assert.EqualValues(t, []string{"pets"}, op.Get.Tags)
	sparam := op.Get.Parameters[0]
	assert.Equal(t, "", sparam.Description)
	assert.Equal(t, "query", sparam.In)
	assert.Equal(t, "string", sparam.Type)
	assert.Equal(t, "", sparam.Format)
	assert.False(t, sparam.Required)
	assert.Equal(t, "Status", sparam.Extensions["x-go-name"])
	assert.Equal(t, "#/responses/genericError", op.Get.Responses.Default.Ref.String())
	rs, ok := op.Get.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.NotNil(t, rs.Schema)
	aprop := rs.Schema
	assert.Equal(t, "array", aprop.Type[0])
	assert.NotNil(t, aprop.Items)
	assert.NotNil(t, aprop.Items.Schema)
	assert.Equal(t, "#/definitions/pet", aprop.Items.Schema.Ref.String())

	// createPet
	assert.NotNil(t, op.Post)
	assert.Equal(t, "Creates a new pet in the store.", op.Post.Summary)
	assert.Equal(t, "", op.Post.Description)
	assert.Equal(t, "createPet", op.Post.ID)
	assert.EqualValues(t, []string{"pets"}, op.Post.Tags)
	verifyRefParam(t, op.Post.Parameters[0], "The pet to submit.", "pet")
	assert.Equal(t, "#/responses/genericError", op.Post.Responses.Default.Ref.String())
	rs, ok = op.Post.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.NotNil(t, rs.Schema)
	aprop = rs.Schema
	assert.Equal(t, "#/definitions/pet", aprop.Ref.String())

	// path /pets/{id}
	op, ok = paths["/pets/{id}"]
	assert.True(t, ok)
	assert.NotNil(t, op)

	// getPetById
	assert.NotNil(t, op.Get)
	assert.Equal(t, "Gets the details for a pet.", op.Get.Summary)
	assert.Equal(t, "", op.Get.Description)
	assert.Equal(t, "getPetById", op.Get.ID)
	assert.EqualValues(t, []string{"pets"}, op.Get.Tags)
	verifyIDParam(t, op.Get.Parameters[0], "The ID of the pet")
	assert.Equal(t, "#/responses/genericError", op.Get.Responses.Default.Ref.String())
	rs, ok = op.Get.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.NotNil(t, rs.Schema)
	aprop = rs.Schema
	assert.Equal(t, "#/definitions/pet", aprop.Ref.String())

	// updatePet
	assert.NotNil(t, op.Put)
	assert.Equal(t, "Updates the details for a pet.", op.Put.Summary)
	assert.Equal(t, "", op.Put.Description)
	assert.Equal(t, "updatePet", op.Put.ID)
	assert.EqualValues(t, []string{"pets"}, op.Put.Tags)
	verifyIDParam(t, op.Put.Parameters[0], "The ID of the pet")
	verifyRefParam(t, op.Put.Parameters[1], "The pet to submit.", "pet")
	assert.Equal(t, "#/responses/genericError", op.Put.Responses.Default.Ref.String())
	rs, ok = op.Put.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.NotNil(t, rs.Schema)
	aprop = rs.Schema
	assert.Equal(t, "#/definitions/pet", aprop.Ref.String())

	// deletePet
	assert.NotNil(t, op.Delete)
	assert.Equal(t, "Deletes a pet from the store.", op.Delete.Summary)
	assert.Equal(t, "", op.Delete.Description)
	assert.Equal(t, "deletePet", op.Delete.ID)
	assert.EqualValues(t, []string{"pets"}, op.Delete.Tags)
	verifyIDParam(t, op.Delete.Parameters[0], "The ID of the pet")
	assert.Equal(t, "#/responses/genericError", op.Delete.Responses.Default.Ref.String())
	_, ok = op.Delete.Responses.StatusCodeResponses[204]
	assert.True(t, ok)

	// path /orders/{id}
	op, ok = paths["/orders/{id}"]
	assert.True(t, ok)
	assert.NotNil(t, op)

	// getOrderDetails
	assert.NotNil(t, op.Get)
	assert.Equal(t, "Gets the details for an order.", op.Get.Summary)
	assert.Equal(t, "", op.Get.Description)
	assert.Equal(t, "getOrderDetails", op.Get.ID)
	assert.EqualValues(t, []string{"orders"}, op.Get.Tags)
	verifyIDParam(t, op.Get.Parameters[0], "The ID of the order")
	assert.Equal(t, "#/responses/genericError", op.Get.Responses.Default.Ref.String())
	rs, ok = op.Get.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.Equal(t, "#/responses/orderResponse", rs.Ref.String())
	rsm := doc.Responses["orderResponse"]
	assert.NotNil(t, rsm.Schema)
	assert.Equal(t, "#/definitions/order", rsm.Schema.Ref.String())

	// cancelOrder
	assert.NotNil(t, op.Delete)
	assert.Equal(t, "Deletes an order.", op.Delete.Summary)
	assert.Equal(t, "", op.Delete.Description)
	assert.Equal(t, "cancelOrder", op.Delete.ID)
	assert.EqualValues(t, []string{"orders"}, op.Delete.Tags)
	verifyIDParam(t, op.Delete.Parameters[0], "The ID of the order")
	assert.Equal(t, "#/responses/genericError", op.Delete.Responses.Default.Ref.String())
	_, ok = op.Delete.Responses.StatusCodeResponses[204]
	assert.True(t, ok)

	// updateOrder
	assert.NotNil(t, op.Put)
	assert.Equal(t, "Updates an order.", op.Put.Summary)
	assert.Equal(t, "", op.Put.Description)
	assert.Equal(t, "updateOrder", op.Put.ID)
	assert.EqualValues(t, []string{"orders"}, op.Put.Tags)
	verifyIDParam(t, op.Put.Parameters[0], "The ID of the order")
	verifyRefParam(t, op.Put.Parameters[1], "The order to submit", "order")
	assert.Equal(t, "#/responses/genericError", op.Put.Responses.Default.Ref.String())
	rs, ok = op.Put.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.NotNil(t, rs.Schema)
	aprop = rs.Schema
	assert.Equal(t, "#/definitions/order", aprop.Ref.String())

	// path /orders
	op, ok = paths["/orders"]
	assert.True(t, ok)
	assert.NotNil(t, op)

	// createOrder
	assert.NotNil(t, op.Post)
	assert.Equal(t, "Creates an order.", op.Post.Summary)
	assert.Equal(t, "", op.Post.Description)
	assert.Equal(t, "createOrder", op.Post.ID)
	assert.EqualValues(t, []string{"orders"}, op.Post.Tags)
	verifyRefParam(t, op.Post.Parameters[0], "The order to submit", "order")
	assert.Equal(t, "#/responses/genericError", op.Post.Responses.Default.Ref.String())
	rs, ok = op.Post.Responses.StatusCodeResponses[200]
	assert.True(t, ok)
	assert.Equal(t, "#/responses/orderResponse", rs.Ref.String())
	rsm = doc.Responses["orderResponse"]
	assert.NotNil(t, rsm.Schema)
	assert.Equal(t, "#/definitions/order", rsm.Schema.Ref.String())
}

func verifyIDParam(t testing.TB, param spec.Parameter, description string) {
	assert.Equal(t, description, param.Description)
	assert.Equal(t, "path", param.In)
	assert.Equal(t, "integer", param.Type)
	assert.Equal(t, "int64", param.Format)
	assert.True(t, param.Required)
	assert.Equal(t, "ID", param.Extensions["x-go-name"])
}

func verifyRefParam(t testing.TB, param spec.Parameter, description, refed string) {
	assert.Equal(t, description, param.Description)
	assert.Equal(t, "body", param.In)
	assert.Equal(t, "#/definitions/"+refed, param.Schema.Ref.String())
	assert.True(t, param.Required)
}

func TestSectionedParser_TitleDescription(t *testing.T) {
	text := `This has a title, separated by a whitespace line

In this example the punctuation for the title should not matter for swagger.
For go it will still make a difference though.
`
	text2 := `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.
`

	text3 := `This has a title, and markdown in the description

See how markdown works now, we can have lists:

+ first item
+ second item
+ third item

[Links works too](http://localhost)
`

	st := &sectionedParser{}
	st.setTitle = func(lines []string) {}
	st.Parse(ascg(text))

	assert.EqualValues(t, []string{"This has a title, separated by a whitespace line"}, st.Title())
	assert.EqualValues(t, []string{"In this example the punctuation for the title should not matter for swagger.", "For go it will still make a difference though."}, st.Description())

	st = &sectionedParser{}
	st.setTitle = func(lines []string) {}
	st.Parse(ascg(text2))

	assert.EqualValues(t, []string{"This has a title without whitespace."}, st.Title())
	assert.EqualValues(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())

	st = &sectionedParser{}
	st.setTitle = func(lines []string) {}
	st.Parse(ascg(text3))

	assert.EqualValues(t, []string{"This has a title, and markdown in the description"}, st.Title())
	assert.EqualValues(t, []string{"See how markdown works now, we can have lists:", "", "+ first item", "+ second item", "+ third item", "", "[Links works too](http://localhost)"}, st.Description())
}

func dummyBuilder() schemaValidations {
	return schemaValidations{new(spec.Schema)}
}

func TestSectionedParser_TagsDescription(t *testing.T) {
	block := `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.
minimum: 10
maximum: 20
`
	block2 := `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
maximum: 20
`

	st := &sectionedParser{}
	st.setTitle = func(lines []string) {}
	st.taggers = []tagParser{
		{"Maximum", false, nil, &setMaximum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, nil, &setMinimum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, nil, &setMultipleOf{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	st.Parse(ascg(block))
	assert.EqualValues(t, []string{"This has a title without whitespace."}, st.Title())
	assert.EqualValues(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 2)
	_, ok := st.matched["Maximum"]
	assert.True(t, ok)
	_, ok = st.matched["Minimum"]
	assert.True(t, ok)

	st = &sectionedParser{}
	st.setTitle = func(lines []string) {}
	st.taggers = []tagParser{
		{"Maximum", false, nil, &setMaximum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, nil, &setMinimum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, nil, &setMultipleOf{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	st.Parse(ascg(block2))
	assert.EqualValues(t, []string{"This has a title without whitespace."}, st.Title())
	assert.EqualValues(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 2)
	_, ok = st.matched["Maximum"]
	assert.True(t, ok)
	_, ok = st.matched["Minimum"]
	assert.True(t, ok)
}

func TestSectionedParser_Empty(t *testing.T) {
	block := `swagger:response someResponse`

	st := &sectionedParser{}
	st.setTitle = func(lines []string) {}
	ap := newSchemaAnnotationParser("SomeResponse")
	ap.rx = rxResponseOverride
	st.annotation = ap

	st.Parse(ascg(block))
	assert.Empty(t, st.Title())
	assert.Empty(t, st.Description())
	assert.Empty(t, st.taggers)
	assert.Equal(t, "SomeResponse", ap.GoName)
	assert.Equal(t, "someResponse", ap.Name)
}

func TestSectionedParser_SkipSectionAnnotation(t *testing.T) {
	block := `swagger:model someModel

This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
maximum: 20
`
	st := &sectionedParser{}
	st.setTitle = func(lines []string) {}
	ap := newSchemaAnnotationParser("SomeModel")
	st.annotation = ap
	st.taggers = []tagParser{
		{"Maximum", false, nil, &setMaximum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, nil, &setMinimum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, nil, &setMultipleOf{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	st.Parse(ascg(block))
	assert.EqualValues(t, []string{"This has a title without whitespace."}, st.Title())
	assert.EqualValues(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 2)
	_, ok := st.matched["Maximum"]
	assert.True(t, ok)
	_, ok = st.matched["Minimum"]
	assert.True(t, ok)
	assert.Equal(t, "SomeModel", ap.GoName)
	assert.Equal(t, "someModel", ap.Name)
}

func TestSectionedParser_TerminateOnNewAnnotation(t *testing.T) {
	block := `swagger:model someModel

This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
swagger:meta
maximum: 20
`
	st := &sectionedParser{}
	st.setTitle = func(lines []string) {}
	ap := newSchemaAnnotationParser("SomeModel")
	st.annotation = ap
	st.taggers = []tagParser{
		{"Maximum", false, nil, &setMaximum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, nil, &setMinimum{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, nil, &setMultipleOf{dummyBuilder(), regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	st.Parse(ascg(block))
	assert.EqualValues(t, []string{"This has a title without whitespace."}, st.Title())
	assert.EqualValues(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 1)
	_, ok := st.matched["Maximum"]
	assert.False(t, ok)
	_, ok = st.matched["Minimum"]
	assert.True(t, ok)
	assert.Equal(t, "SomeModel", ap.GoName)
	assert.Equal(t, "someModel", ap.Name)
}

func ascg(txt string) *ast.CommentGroup {
	var cg ast.CommentGroup
	for _, line := range strings.Split(txt, "\n") {
		var cmt ast.Comment
		cmt.Text = "// " + line
		cg.List = append(cg.List, &cmt)
	}
	return &cg
}

func TestSchemaValueExtractors(t *testing.T) {
	strfmts := []string{
		"// swagger:strfmt ",
		"* swagger:strfmt ",
		"* swagger:strfmt ",
		" swagger:strfmt ",
		"swagger:strfmt ",
		"// swagger:strfmt    ",
		"* swagger:strfmt     ",
		"* swagger:strfmt    ",
		" swagger:strfmt     ",
		"swagger:strfmt      ",
	}
	models := []string{
		"// swagger:model ",
		"* swagger:model ",
		"* swagger:model ",
		" swagger:model ",
		"swagger:model ",
		"// swagger:model    ",
		"* swagger:model     ",
		"* swagger:model    ",
		" swagger:model     ",
		"swagger:model      ",
	}

	allOf := []string{
		"// swagger:allOf ",
		"* swagger:allOf ",
		"* swagger:allOf ",
		" swagger:allOf ",
		"swagger:allOf ",
		"// swagger:allOf    ",
		"* swagger:allOf     ",
		"* swagger:allOf    ",
		" swagger:allOf     ",
		"swagger:allOf      ",
	}

	discriminated := []string{
		"// swagger:discriminated ",
		"* swagger:discriminated ",
		"* swagger:discriminated ",
		" swagger:discriminated ",
		"swagger:discriminated ",
		"// swagger:discriminated    ",
		"* swagger:discriminated     ",
		"* swagger:discriminated    ",
		" swagger:discriminated     ",
		"swagger:discriminated      ",
	}

	parameters := []string{
		"// swagger:parameters ",
		"* swagger:parameters ",
		"* swagger:parameters ",
		" swagger:parameters ",
		"swagger:parameters ",
		"// swagger:parameters    ",
		"* swagger:parameters     ",
		"* swagger:parameters    ",
		" swagger:parameters     ",
		"swagger:parameters      ",
	}
	validParams := []string{
		"yada123",
		"date",
		"date-time",
		"long-combo-1-with-combo-2-and-a-3rd-one-too",
	}
	invalidParams := []string{
		"1-yada-3",
		"1-2-3",
		"-yada-3",
		"-2-3",
		"*blah",
		"blah*",
	}

	verifySwaggerOneArgSwaggerTag(t, rxStrFmt, strfmts, validParams, append(invalidParams, "", "  ", " "))
	verifySwaggerOneArgSwaggerTag(t, rxModelOverride, models, append(validParams, "", "  ", " "), invalidParams)

	verifySwaggerOneArgSwaggerTag(t, rxAllOf, allOf, append(validParams, "", "  ", " "), invalidParams)
	verifySwaggerMultiArgSwaggerTag(t, rxDiscriminated, discriminated, validParams, invalidParams)

	verifySwaggerMultiArgSwaggerTag(t, rxParametersOverride, parameters, validParams, invalidParams)

	verifyMinMax(t, rxf(rxMinimumFmt, ""), "min", []string{"", ">", "="})
	verifyMinMax(t, rxf(rxMinimumFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.min", []string{"", ">", "="})
	verifyMinMax(t, rxf(rxMaximumFmt, ""), "max", []string{"", "<", "="})
	verifyMinMax(t, rxf(rxMaximumFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.max", []string{"", "<", "="})
	verifyNumeric2Words(t, rxf(rxMultipleOfFmt, ""), "multiple", "of")
	verifyNumeric2Words(t, rxf(rxMultipleOfFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.multiple", "of")

	verifyIntegerMinMaxManyWords(t, rxf(rxMinLengthFmt, ""), "min", []string{"len", "length"})
	// pattern
	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	patArgs := []string{"^\\w+$", "[A-Za-z0-9-.]*"}
	patNames := []string{"pattern", "Pattern"}
	for _, pref := range prefixes {
		for _, es1 := range extraSpaces {
			for _, nm := range patNames {
				for _, es2 := range extraSpaces {
					for _, es3 := range extraSpaces {
						for _, arg := range patArgs {
							line := strings.Join([]string{pref, es1, nm, es2, ":", es3, arg}, "")
							matches := rxf(rxPatternFmt, "").FindStringSubmatch(line)
							assert.Len(t, matches, 2)
							assert.Equal(t, arg, matches[1])
						}
					}
				}
			}
		}
	}

	verifyIntegerMinMaxManyWords(t, rxf(rxMinItemsFmt, ""), "min", []string{"items"})
	verifyBoolean(t, rxf(rxUniqueFmt, ""), []string{"unique"}, nil)

	verifyBoolean(t, rxReadOnly, []string{"read"}, []string{"only"})
	verifyBoolean(t, rxRequired, []string{"required"}, nil)
}

func makeMinMax(lower string) (res []string) {
	for _, a := range []string{"", "imum"} {
		res = append(res, lower+a, strings.Title(lower)+a)
	}
	return
}

func verifyBoolean(t *testing.T, matcher *regexp.Regexp, names, names2 []string) {
	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validArgs := []string{"true", "false"}
	invalidArgs := []string{"TRUE", "FALSE", "t", "f", "1", "0", "True", "False", "true*", "false*"}
	var nms []string
	for _, nm := range names {
		nms = append(nms, nm, strings.Title(nm))
	}

	var nms2 []string
	for _, nm := range names2 {
		nms2 = append(nms2, nm, strings.Title(nm))
	}

	var rnms []string
	if len(nms2) > 0 {
		for _, nm := range nms {
			for _, es := range append(extraSpaces, "-") {
				for _, nm2 := range nms2 {
					rnms = append(rnms, strings.Join([]string{nm, es, nm2}, ""))
				}
			}
		}
	} else {
		rnms = nms
	}

	var cnt int
	for _, pref := range prefixes {
		for _, es1 := range extraSpaces {
			for _, nm := range rnms {
				for _, es2 := range extraSpaces {
					for _, es3 := range extraSpaces {
						for _, vv := range validArgs {
							line := strings.Join([]string{pref, es1, nm, es2, ":", es3, vv}, "")
							matches := matcher.FindStringSubmatch(line)
							assert.Len(t, matches, 2)
							assert.Equal(t, vv, matches[1])
							cnt++
						}
						for _, iv := range invalidArgs {
							line := strings.Join([]string{pref, es1, nm, es2, ":", es3, iv}, "")
							matches := matcher.FindStringSubmatch(line)
							assert.Empty(t, matches)
							cnt++
						}
					}
				}
			}
		}
	}
	var nm2 string
	if len(names2) > 0 {
		nm2 = " " + names2[0]
	}
	fmt.Printf("tested %d %s%s combinations\n", cnt, names[0], nm2)
}

func verifyIntegerMinMaxManyWords(t *testing.T, matcher *regexp.Regexp, name1 string, words []string) {
	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validNumericArgs := []string{"0", "1234"}
	invalidNumericArgs := []string{"1A3F", "2e10", "*12", "12*", "-1235", "0.0", "1234.0394", "-2948.484"}

	var names []string
	for _, w := range words {
		names = append(names, w, strings.Title(w))
	}

	var cnt int
	for _, pref := range prefixes {
		for _, es1 := range extraSpaces {
			for _, nm1 := range makeMinMax(name1) {
				for _, es2 := range append(extraSpaces, "-") {
					for _, nm2 := range names {
						for _, es3 := range extraSpaces {
							for _, es4 := range extraSpaces {
								for _, vv := range validNumericArgs {
									line := strings.Join([]string{pref, es1, nm1, es2, nm2, es3, ":", es4, vv}, "")
									matches := matcher.FindStringSubmatch(line)
									//fmt.Printf("matching %q, matches (%d): %v\n", line, len(matches), matches)
									assert.Len(t, matches, 2)
									assert.Equal(t, vv, matches[1])
									cnt++
								}
								for _, iv := range invalidNumericArgs {
									line := strings.Join([]string{pref, es1, nm1, es2, nm2, es3, ":", es4, iv}, "")
									matches := matcher.FindStringSubmatch(line)
									assert.Empty(t, matches)
									cnt++
								}
							}
						}
					}
				}
			}
		}
	}
	var nm2 string
	if len(words) > 0 {
		nm2 = " " + words[0]
	}
	fmt.Printf("tested %d %s%s combinations\n", cnt, name1, nm2)
}

func verifyNumeric2Words(t *testing.T, matcher *regexp.Regexp, name1, name2 string) {
	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validNumericArgs := []string{"0", "1234", "-1235", "0.0", "1234.0394", "-2948.484"}
	invalidNumericArgs := []string{"1A3F", "2e10", "*12", "12*"}

	var cnt int
	for _, pref := range prefixes {
		for _, es1 := range extraSpaces {
			for _, es2 := range extraSpaces {
				for _, es3 := range extraSpaces {
					for _, es4 := range extraSpaces {
						for _, vv := range validNumericArgs {
							lines := []string{
								strings.Join([]string{pref, es1, name1, es2, name2, es3, ":", es4, vv}, ""),
								strings.Join([]string{pref, es1, strings.Title(name1), es2, strings.Title(name2), es3, ":", es4, vv}, ""),
								strings.Join([]string{pref, es1, strings.Title(name1), es2, name2, es3, ":", es4, vv}, ""),
								strings.Join([]string{pref, es1, name1, es2, strings.Title(name2), es3, ":", es4, vv}, ""),
							}
							for _, line := range lines {
								matches := matcher.FindStringSubmatch(line)
								//fmt.Printf("matching %q, matches (%d): %v\n", line, len(matches), matches)
								assert.Len(t, matches, 2)
								assert.Equal(t, vv, matches[1])
								cnt++
							}
						}
						for _, iv := range invalidNumericArgs {
							lines := []string{
								strings.Join([]string{pref, es1, name1, es2, name2, es3, ":", es4, iv}, ""),
								strings.Join([]string{pref, es1, strings.Title(name1), es2, strings.Title(name2), es3, ":", es4, iv}, ""),
								strings.Join([]string{pref, es1, strings.Title(name1), es2, name2, es3, ":", es4, iv}, ""),
								strings.Join([]string{pref, es1, name1, es2, strings.Title(name2), es3, ":", es4, iv}, ""),
							}
							for _, line := range lines {
								matches := matcher.FindStringSubmatch(line)
								//fmt.Printf("matching %q, matches (%d): %v\n", line, len(matches), matches)
								assert.Empty(t, matches)
								cnt++
							}
						}
					}
				}
			}
		}
	}
	fmt.Printf("tested %d %s %s combinations\n", cnt, name1, name2)
}

func verifyMinMax(t *testing.T, matcher *regexp.Regexp, name string, operators []string) {
	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validNumericArgs := []string{"0", "1234", "-1235", "0.0", "1234.0394", "-2948.484"}
	invalidNumericArgs := []string{"1A3F", "2e10", "*12", "12*"}

	var cnt int
	for _, pref := range prefixes {
		for _, es1 := range extraSpaces {
			for _, wrd := range makeMinMax(name) {
				for _, es2 := range extraSpaces {
					for _, es3 := range extraSpaces {
						for _, op := range operators {
							for _, es4 := range extraSpaces {
								for _, vv := range validNumericArgs {
									line := strings.Join([]string{pref, es1, wrd, es2, ":", es3, op, es4, vv}, "")
									matches := matcher.FindStringSubmatch(line)
									// fmt.Printf("matching %q with %q, matches (%d): %v\n", line, matcher, len(matches), matches)
									assert.Len(t, matches, 3)
									assert.Equal(t, vv, matches[2])
									cnt++
								}
								for _, iv := range invalidNumericArgs {
									line := strings.Join([]string{pref, es1, wrd, es2, ":", es3, op, es4, iv}, "")
									matches := matcher.FindStringSubmatch(line)
									assert.Empty(t, matches)
									cnt++
								}
							}
						}
					}
				}
			}
		}
	}
	fmt.Printf("tested %d %s combinations\n", cnt, name)
}

func verifySwaggerOneArgSwaggerTag(t *testing.T, matcher *regexp.Regexp, prefixes, validParams, invalidParams []string) {
	for _, pref := range prefixes {
		for _, param := range validParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			if assert.Len(t, matches, 2) {
				assert.Equal(t, strings.TrimSpace(param), matches[1])
			}
		}
	}

	for _, pref := range prefixes {
		for _, param := range invalidParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			assert.Empty(t, matches)
		}
	}
}

func verifySwaggerMultiArgSwaggerTag(t *testing.T, matcher *regexp.Regexp, prefixes, validParams, invalidParams []string) {
	var actualParams []string
	for i := 0; i < len(validParams); i++ {
		var vp []string
		for j := 0; j < (i + 1); j++ {
			vp = append(vp, validParams[j])
		}
		actualParams = append(actualParams, strings.Join(vp, " "))
	}
	for _, pref := range prefixes {
		for _, param := range actualParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			// fmt.Printf("matching %q with %q, matches (%d): %v\n", line, matcher, len(matches), matches)
			assert.Len(t, matches, 2)
			assert.Equal(t, strings.TrimSpace(param), matches[1])
		}
	}

	for _, pref := range prefixes {
		for _, param := range invalidParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			assert.Empty(t, matches)
		}
	}
}
