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
	"path/filepath"
	"testing"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/stretchr/testify/assert"
)

func TestSchemaParser(t *testing.T) {
	_ = classificationProg
	schema := noModelDefs["NoModel"]

	assert.Equal(t, spec.StringOrArray([]string{"object"}), schema.Type)
	assert.Equal(t, "NoModel is a struct without an annotation.", schema.Title)
	assert.Equal(t, "NoModel exists in a package\nbut is not annotated with the swagger model annotations\nso it should now show up in a test.", schema.Description)
	assert.Len(t, schema.Required, 3)
	assert.Len(t, schema.Properties, 7)

	assertProperty(t, &schema, "integer", "id", "int64", "ID")
	prop, ok := schema.Properties["id"]
	assert.Equal(t, "ID of this no model instance.\nids in this application start at 11 and are smaller than 1000", prop.Description)
	assert.True(t, ok, "should have had an 'id' property")
	assert.EqualValues(t, 1000, *prop.Maximum)
	assert.True(t, prop.ExclusiveMaximum, "'id' should have had an exclusive maximum")
	assert.NotNil(t, prop.Minimum)
	assert.EqualValues(t, 10, *prop.Minimum)
	assert.True(t, prop.ExclusiveMinimum, "'id' should have had an exclusive minimum")

	assertProperty(t, &schema, "integer", "score", "int32", "Score")
	prop, ok = schema.Properties["score"]
	assert.Equal(t, "The Score of this model", prop.Description)
	assert.True(t, ok, "should have had a 'score' property")
	assert.EqualValues(t, 45, *prop.Maximum)
	assert.False(t, prop.ExclusiveMaximum, "'score' should not have had an exclusive maximum")
	assert.NotNil(t, prop.Minimum)
	assert.EqualValues(t, 3, *prop.Minimum)
	assert.False(t, prop.ExclusiveMinimum, "'score' should not have had an exclusive minimum")

	assertProperty(t, &schema, "string", "name", "", "Name")
	prop, ok = schema.Properties["name"]
	assert.Equal(t, "Name of this no model instance", prop.Description)
	assert.EqualValues(t, 4, *prop.MinLength)
	assert.EqualValues(t, 50, *prop.MaxLength)
	assert.Equal(t, "[A-Za-z0-9-.]*", prop.Pattern)

	assertProperty(t, &schema, "string", "created", "date-time", "Created")
	prop, ok = schema.Properties["created"]
	assert.Equal(t, "Created holds the time when this entry was created", prop.Description)
	assert.True(t, ok, "should have a 'created' property")
	assert.True(t, prop.ReadOnly, "'created' should be read only")

	assertArrayProperty(t, &schema, "string", "foo_slice", "", "FooSlice")
	prop, ok = schema.Properties["foo_slice"]
	assert.Equal(t, "a FooSlice has foos which are strings", prop.Description)
	assert.True(t, ok, "should have a 'foo_slice' property")
	assert.NotNil(t, prop.Items, "foo_slice should have had an items property")
	assert.NotNil(t, prop.Items.Schema, "foo_slice.items should have had a schema property")
	assert.True(t, prop.UniqueItems, "'foo_slice' should have unique items")
	assert.EqualValues(t, 3, *prop.MinItems, "'foo_slice' should have had 3 min items")
	assert.EqualValues(t, 10, *prop.MaxItems, "'foo_slice' should have had 10 max items")
	itprop := prop.Items.Schema
	assert.EqualValues(t, 3, *itprop.MinLength, "'foo_slice.items.minLength' should have been 3")
	assert.EqualValues(t, 10, *itprop.MaxLength, "'foo_slice.items.maxLength' should have been 10")
	assert.EqualValues(t, "\\w+", itprop.Pattern, "'foo_slice.items.pattern' should have \\w+")

	assertArrayProperty(t, &schema, "array", "bar_slice", "", "BarSlice")
	prop, ok = schema.Properties["bar_slice"]
	assert.Equal(t, "a BarSlice has bars which are strings", prop.Description)
	assert.True(t, ok, "should have a 'bar_slice' property")
	assert.NotNil(t, prop.Items, "bar_slice should have had an items property")
	assert.NotNil(t, prop.Items.Schema, "bar_slice.items should have had a schema property")
	assert.True(t, prop.UniqueItems, "'bar_slice' should have unique items")
	assert.EqualValues(t, 3, *prop.MinItems, "'bar_slice' should have had 3 min items")
	assert.EqualValues(t, 10, *prop.MaxItems, "'bar_slice' should have had 10 max items")
	itprop = prop.Items.Schema
	if assert.NotNil(t, itprop) {
		assert.EqualValues(t, 4, *itprop.MinItems, "'bar_slice.items.minItems' should have been 4")
		assert.EqualValues(t, 9, *itprop.MaxItems, "'bar_slice.items.maxItems' should have been 9")
		itprop2 := itprop.Items.Schema
		if assert.NotNil(t, itprop2) {
			assert.EqualValues(t, 5, *itprop2.MinItems, "'bar_slice.items.items.minItems' should have been 5")
			assert.EqualValues(t, 8, *itprop2.MaxItems, "'bar_slice.items.items.maxItems' should have been 8")
			itprop3 := itprop2.Items.Schema
			if assert.NotNil(t, itprop3) {
				assert.EqualValues(t, 3, *itprop3.MinLength, "'bar_slice.items.items.items.minLength' should have been 3")
				assert.EqualValues(t, 10, *itprop3.MaxLength, "'bar_slice.items.items.items.maxLength' should have been 10")
				assert.EqualValues(t, "\\w+", itprop3.Pattern, "'bar_slice.items.items.items.pattern' should have \\w+")
			}
		}
	}

	assertArrayProperty(t, &schema, "object", "items", "", "Items")
	prop, ok = schema.Properties["items"]
	assert.True(t, ok, "should have an 'items' slice")
	assert.NotNil(t, prop.Items, "items should have had an items property")
	assert.NotNil(t, prop.Items.Schema, "items.items should have had a schema property")
	itprop = prop.Items.Schema
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

	definitions := make(map[string]spec.Schema)
	sp := newSchemaParser(classificationProg)
	pn := "github.com/go-swagger/go-swagger/fixtures/goparsing/classification/models"
	// pnr := "../fixtures/goparsing/classification/models"
	pkg := classificationProg.Package(pn)
	if assert.NotNil(t, pkg) {

		fnd := false
		for _, fil := range pkg.Files {
			nm := filepath.Base(classificationProg.Fset.File(fil.Pos()).Name())
			if nm == "order.go" {
				fnd = true
				sp.Parse(fil, definitions)
				break
			}
		}
		assert.True(t, fnd)
		msch, ok := definitions["order"]
		assert.True(t, ok)
		assert.Equal(t, pn, msch.Extensions["x-go-package"])
		assert.Equal(t, "StoreOrder", msch.Extensions["x-go-name"])
	}
}

func TestEmbeddedTypes(t *testing.T) {
	schema := noModelDefs["ComplexerOne"]
	assertProperty(t, &schema, "integer", "age", "int32", "Age")
	assertProperty(t, &schema, "integer", "id", "int64", "ID")
	assertProperty(t, &schema, "string", "createdAt", "date-time", "CreatedAt")
	assertProperty(t, &schema, "string", "extra", "", "Extra")
	assertProperty(t, &schema, "string", "name", "", "Name")
	assertProperty(t, &schema, "string", "notes", "", "Notes")
}

func TestEmbeddedAllOf(t *testing.T) {
	schema := noModelDefs["AllOfModel"]

	assert.Len(t, schema.AllOf, 3)
	asch := schema.AllOf[0]
	assertProperty(t, &asch, "integer", "age", "int32", "Age")
	assertProperty(t, &asch, "integer", "id", "int64", "ID")
	assertProperty(t, &asch, "string", "name", "", "Name")

	asch = schema.AllOf[1]
	assert.Equal(t, "#/definitions/withNotes", asch.Ref.String())

	asch = schema.AllOf[2]
	assertProperty(t, &asch, "string", "createdAt", "date-time", "CreatedAt")
	assertProperty(t, &asch, "integer", "did", "int64", "DID")
	assertProperty(t, &asch, "string", "cat", "", "Cat")
}

func TestAliasedTypes(t *testing.T) {
	schema := noModelDefs["OtherTypes"]
	assertProperty(t, &schema, "string", "named", "", "Named")
	assertProperty(t, &schema, "integer", "numbered", "int64", "Numbered")
	assertProperty(t, &schema, "string", "timed", "date-time", "Timed")
	assertRef(t, &schema, "petted", "Petted", "#/definitions/pet")
	assertRef(t, &schema, "somethinged", "Somethinged", "#/definitions/Something")
	assertProperty(t, &schema, "string", "dated", "date-time", "Dated")

	assertArrayProperty(t, &schema, "string", "manyNamed", "", "ManyNamed")
	assertArrayProperty(t, &schema, "integer", "manyNumbered", "int64", "ManyNumbered")
	assertArrayProperty(t, &schema, "string", "manyTimed", "date-time", "ManyTimed")
	assertArrayRef(t, &schema, "manyPetted", "ManyPetted", "#/definitions/pet")
	assertArrayRef(t, &schema, "manySomethinged", "ManySomethinged", "#/definitions/Something")
	assertArrayProperty(t, &schema, "string", "manyDated", "date-time", "ManyDated")

	assertArrayProperty(t, &schema, "string", "nameds", "", "Nameds")
	assertArrayProperty(t, &schema, "integer", "numbereds", "int64", "Numbereds")
	assertArrayProperty(t, &schema, "string", "timeds", "date-time", "Timeds")
	assertArrayRef(t, &schema, "petteds", "Petteds", "#/definitions/pet")
	assertArrayRef(t, &schema, "somethingeds", "Somethingeds", "#/definitions/Something")
	assertArrayProperty(t, &schema, "string", "dateds", "date-time", "Dateds")

	assertProperty(t, &schema, "string", "modsNamed", "", "ModsNamed")
	assertProperty(t, &schema, "integer", "modsNumbered", "int64", "ModsNumbered")
	assertProperty(t, &schema, "string", "modsTimed", "date-time", "ModsTimed")
	assertRef(t, &schema, "modsPetted", "ModsPetted", "#/definitions/pet")
	assertProperty(t, &schema, "string", "modsDated", "date-time", "ModsDated")

	assertArrayProperty(t, &schema, "string", "manyModsNamed", "", "ManyModsNamed")
	assertArrayProperty(t, &schema, "integer", "manyModsNumbered", "int64", "ManyModsNumbered")
	assertArrayProperty(t, &schema, "string", "manyModsTimed", "date-time", "ManyModsTimed")
	assertArrayRef(t, &schema, "manyModsPetted", "ManyModsPetted", "#/definitions/pet")
	assertArrayProperty(t, &schema, "string", "manyModsDated", "date-time", "ManyModsDated")

	assertArrayProperty(t, &schema, "string", "modsNameds", "", "ModsNameds")
	assertArrayProperty(t, &schema, "integer", "modsNumbereds", "int64", "ModsNumbereds")
	assertArrayProperty(t, &schema, "string", "modsTimeds", "date-time", "ModsTimeds")
	assertArrayRef(t, &schema, "modsPetteds", "ModsPetteds", "#/definitions/pet")
	assertArrayProperty(t, &schema, "string", "modsDateds", "date-time", "ModsDateds")
}

func TestParsePrimitiveSchemaProperty(t *testing.T) {
	schema := noModelDefs["PrimateModel"]
	assertProperty(t, &schema, "boolean", "a", "", "A")
	assertProperty(t, &schema, "string", "b", "", "B")
	assertProperty(t, &schema, "string", "c", "", "C")
	assertProperty(t, &schema, "integer", "d", "int64", "D")
	assertProperty(t, &schema, "integer", "e", "int8", "E")
	assertProperty(t, &schema, "integer", "f", "int16", "F")
	assertProperty(t, &schema, "integer", "g", "int32", "G")
	assertProperty(t, &schema, "integer", "h", "int64", "H")
	assertProperty(t, &schema, "integer", "i", "uint64", "I")
	assertProperty(t, &schema, "integer", "j", "uint8", "J")
	assertProperty(t, &schema, "integer", "k", "uint16", "K")
	assertProperty(t, &schema, "integer", "l", "uint32", "L")
	assertProperty(t, &schema, "integer", "m", "uint64", "M")
	assertProperty(t, &schema, "number", "n", "float", "N")
	assertProperty(t, &schema, "number", "o", "double", "O")
}

func TestParseStringFormatSchemaProperty(t *testing.T) {
	schema := noModelDefs["FormattedModel"]
	assertProperty(t, &schema, "string", "a", "byte", "A")
	assertProperty(t, &schema, "string", "b", "creditcard", "B")
	assertProperty(t, &schema, "string", "c", "date", "C")
	assertProperty(t, &schema, "string", "d", "date-time", "D")
	assertProperty(t, &schema, "string", "e", "duration", "E")
	assertProperty(t, &schema, "string", "f", "email", "F")
	assertProperty(t, &schema, "string", "g", "hexcolor", "G")
	assertProperty(t, &schema, "string", "h", "hostname", "H")
	assertProperty(t, &schema, "string", "i", "ipv4", "I")
	assertProperty(t, &schema, "string", "j", "ipv6", "J")
	assertProperty(t, &schema, "string", "k", "isbn", "K")
	assertProperty(t, &schema, "string", "l", "isbn10", "L")
	assertProperty(t, &schema, "string", "m", "isbn13", "M")
	assertProperty(t, &schema, "string", "n", "rgbcolor", "N")
	assertProperty(t, &schema, "string", "o", "ssn", "O")
	assertProperty(t, &schema, "string", "p", "uri", "P")
	assertProperty(t, &schema, "string", "q", "uuid", "Q")
	assertProperty(t, &schema, "string", "r", "uuid3", "R")
	assertProperty(t, &schema, "string", "s", "uuid4", "S")
	assertProperty(t, &schema, "string", "t", "uuid5", "T")
}

func assertProperty(t testing.TB, schema *spec.Schema, typeName, jsonName, format, goName string) {
	if typeName == "" {
		assert.Empty(t, schema.Properties[jsonName].Type)
	} else {
		if assert.NotEmpty(t, schema.Properties[jsonName].Type) {
			assert.Equal(t, typeName, schema.Properties[jsonName].Type[0])
		}
	}
	assert.Equal(t, goName, schema.Properties[jsonName].Extensions["x-go-name"])
	assert.Equal(t, format, schema.Properties[jsonName].Format)
}

func assertRef(t testing.TB, schema *spec.Schema, jsonName, goName, fragment string) {
	assertProperty(t, schema, "", jsonName, "", goName)
	psch := schema.Properties[jsonName]
	assert.Equal(t, fragment, psch.Ref.String())
}

func TestParseStructFields(t *testing.T) {
	schema := noModelDefs["SimpleComplexModel"]
	assertProperty(t, &schema, "object", "emb", "", "Emb")
	eSchema := schema.Properties["emb"]
	assertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	assertProperty(t, &eSchema, "string", "baz", "", "Baz")

	assertRef(t, &schema, "top", "Top", "#/definitions/Something")
	assertRef(t, &schema, "notSel", "NotSel", "#/definitions/NotSelected")
}

func TestParsePointerFields(t *testing.T) {
	schema := noModelDefs["Pointdexter"]

	assertProperty(t, &schema, "integer", "id", "int64", "ID")
	assertProperty(t, &schema, "string", "name", "", "Name")
	assertProperty(t, &schema, "object", "emb", "", "Emb")
	assertProperty(t, &schema, "string", "t", "uuid5", "T")
	eSchema := schema.Properties["emb"]
	assertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	assertProperty(t, &eSchema, "string", "baz", "", "Baz")

	assertRef(t, &schema, "top", "Top", "#/definitions/Something")
	assertRef(t, &schema, "notSel", "NotSel", "#/definitions/NotSelected")
}

func assertArrayProperty(t testing.TB, schema *spec.Schema, typeName, jsonName, format, goName string) {
	prop := schema.Properties[jsonName]
	assert.NotEmpty(t, prop.Type)
	assert.True(t, prop.Type.Contains("array"))
	assert.NotNil(t, prop.Items)
	if typeName != "" {
		assert.Equal(t, typeName, prop.Items.Schema.Type[0])
	}
	assert.Equal(t, goName, prop.Extensions["x-go-name"])
	assert.Equal(t, format, prop.Items.Schema.Format)
}

func assertArrayRef(t testing.TB, schema *spec.Schema, jsonName, goName, fragment string) {
	assertArrayProperty(t, schema, "", jsonName, "", goName)
	psch := schema.Properties[jsonName].Items.Schema
	assert.Equal(t, fragment, psch.Ref.String())
}

func TestParseSliceFields(t *testing.T) {
	schema := noModelDefs["SliceAndDice"]

	assertArrayProperty(t, &schema, "integer", "ids", "int64", "IDs")
	assertArrayProperty(t, &schema, "string", "names", "", "Names")
	assertArrayProperty(t, &schema, "string", "uuids", "uuid", "UUIDs")
	assertArrayProperty(t, &schema, "object", "embs", "", "Embs")
	eSchema := schema.Properties["embs"].Items.Schema
	assertArrayProperty(t, eSchema, "integer", "cid", "int64", "CID")
	assertArrayProperty(t, eSchema, "string", "baz", "", "Baz")

	assertArrayRef(t, &schema, "tops", "Tops", "#/definitions/Something")
	assertArrayRef(t, &schema, "notSels", "NotSels", "#/definitions/NotSelected")

	assertArrayProperty(t, &schema, "integer", "ptrIds", "int64", "PtrIDs")
	assertArrayProperty(t, &schema, "string", "ptrNames", "", "PtrNames")
	assertArrayProperty(t, &schema, "string", "ptrUuids", "uuid", "PtrUUIDs")
	assertArrayProperty(t, &schema, "object", "ptrEmbs", "", "PtrEmbs")
	eSchema = schema.Properties["ptrEmbs"].Items.Schema
	assertArrayProperty(t, eSchema, "integer", "ptrCid", "int64", "PtrCID")
	assertArrayProperty(t, eSchema, "string", "ptrBaz", "", "PtrBaz")

	assertArrayRef(t, &schema, "ptrTops", "PtrTops", "#/definitions/Something")
	assertArrayRef(t, &schema, "ptrNotSels", "PtrNotSels", "#/definitions/NotSelected")
}

func assertMapProperty(t testing.TB, schema *spec.Schema, typeName, jsonName, format, goName string) {
	prop := schema.Properties[jsonName]
	assert.NotEmpty(t, prop.Type)
	assert.True(t, prop.Type.Contains("object"))
	assert.NotNil(t, prop.AdditionalProperties)
	if typeName != "" {
		assert.Equal(t, typeName, prop.AdditionalProperties.Schema.Type[0])
	}
	assert.Equal(t, goName, prop.Extensions["x-go-name"])
	assert.Equal(t, format, prop.AdditionalProperties.Schema.Format)
}

func assertMapRef(t testing.TB, schema *spec.Schema, jsonName, goName, fragment string) {
	assertMapProperty(t, schema, "", jsonName, "", goName)
	psch := schema.Properties[jsonName].AdditionalProperties.Schema
	assert.Equal(t, fragment, psch.Ref.String())
}

func TestParseMapFields(t *testing.T) {
	schema := noModelDefs["MapTastic"]

	assertMapProperty(t, &schema, "integer", "ids", "int64", "IDs")
	assertMapProperty(t, &schema, "string", "names", "", "Names")
	assertMapProperty(t, &schema, "string", "uuids", "uuid", "UUIDs")
	assertMapProperty(t, &schema, "object", "embs", "", "Embs")
	eSchema := schema.Properties["embs"].AdditionalProperties.Schema
	assertMapProperty(t, eSchema, "integer", "cid", "int64", "CID")
	assertMapProperty(t, eSchema, "string", "baz", "", "Baz")

	assertMapRef(t, &schema, "tops", "Tops", "#/definitions/Something")
	assertMapRef(t, &schema, "notSels", "NotSels", "#/definitions/NotSelected")

	assertMapProperty(t, &schema, "integer", "ptrIds", "int64", "PtrIDs")
	assertMapProperty(t, &schema, "string", "ptrNames", "", "PtrNames")
	assertMapProperty(t, &schema, "string", "ptrUuids", "uuid", "PtrUUIDs")
	assertMapProperty(t, &schema, "object", "ptrEmbs", "", "PtrEmbs")
	eSchema = schema.Properties["ptrEmbs"].AdditionalProperties.Schema
	assertMapProperty(t, eSchema, "integer", "ptrCid", "int64", "PtrCID")
	assertMapProperty(t, eSchema, "string", "ptrBaz", "", "PtrBaz")

	assertMapRef(t, &schema, "ptrTops", "PtrTops", "#/definitions/Something")
	assertMapRef(t, &schema, "ptrNotSels", "PtrNotSels", "#/definitions/NotSelected")
}

func TestInterfaceField(t *testing.T) {

	_ = classificationProg
	schema := noModelDefs["Interfaced"]
	assertProperty(t, &schema, "object", "custom_data", "", "CustomData")
}

func TestStructDiscriminators(t *testing.T) {
	_ = classificationProg
	schema := noModelDefs["animal"]

	assert.Equal(t, "BaseStruct", schema.Extensions["x-go-name"])
	assert.Equal(t, schema.Discriminator, "jsonClass")

	sch := noModelDefs["gazelle"]
	assert.Len(t, sch.AllOf, 2)
	cl, _ := sch.Extensions.GetString("x-class")
	assert.Equal(t, "a.b.c.d.E", cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.Equal(t, "Gazelle", cl)

	sch = noModelDefs["giraffe"]
	assert.Len(t, sch.AllOf, 2)
	cl, _ = sch.Extensions.GetString("x-class")
	assert.Equal(t, "", cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.Equal(t, "Giraffe", cl)

	//sch = noModelDefs["lion"]

	//b, _ := json.MarshalIndent(sch, "", "  ")
	//fmt.Println(string(b))

}

func TestInterfaceDiscriminators(t *testing.T) {
	_ = classificationProg
	schema, ok := noModelDefs["fish"]
	if assert.True(t, ok) && assert.Len(t, schema.AllOf, 5) {
		sch := schema.AllOf[0]
		assert.Len(t, sch.Properties, 1)
		assertProperty(t, &sch, "integer", "id", "int64", "ID")

		sch = schema.AllOf[1]
		assert.Equal(t, "#/definitions/water", sch.Ref.String())
		sch = schema.AllOf[2]
		assert.Equal(t, "#/definitions/extra", sch.Ref.String())

		sch = schema.AllOf[3]
		assert.Len(t, sch.Properties, 1)
		assertProperty(t, &sch, "string", "colorName", "", "ColorName")

		sch = schema.AllOf[4]
		assert.Len(t, sch.Properties, 2)
		assertProperty(t, &sch, "string", "name", "", "Name")
		assertProperty(t, &sch, "string", "jsonClass", "", "StructType")
		assert.Equal(t, "jsonClass", sch.Discriminator)
	}

	schema, ok = noModelDefs["modelS"]
	if assert.True(t, ok) {
		assert.Len(t, schema.AllOf, 2)
		cl, _ := schema.Extensions.GetString("x-class")
		assert.Equal(t, "com.tesla.models.ModelS", cl)
		cl, _ = schema.Extensions.GetString("x-go-name")
		assert.Equal(t, "ModelS", cl)

		sch := schema.AllOf[0]
		assert.Equal(t, "#/definitions/TeslaCar", sch.Ref.String())
		sch = schema.AllOf[1]
		assert.Len(t, sch.Properties, 1)
		assertProperty(t, &sch, "string", "edition", "", "Edition")
	}
}
