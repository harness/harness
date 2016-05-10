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
	"strings"
	"testing"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/stretchr/testify/assert"
)

func TestEnum_StringThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "StringThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("string_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var stringThingEnum []interface{}", res)
					assertInCode(t, k+") validateStringThingEnum(path, location string, value StringThing)", res)
					assertInCode(t, "m.validateStringThingEnum(\"\", \"body\", m)", res)
				}
			}
		}
	}
}

func TestEnum_ComposedThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "ComposedThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("composed_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "m.StringThing.Validate(formats)", res)
					assertInCode(t, "var composedThingTypeNamePropEnum []interface{}", res)
					assertInCode(t, "m.validateNameEnum(\"name\", \"body\", m.Name)", res)
					assertInCode(t, k+") validateNameEnum(path, location string, value string)", res)
				}
			}
		}
	}
}

func TestEnum_IntThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "IntThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("int_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var intThingEnum []interface{}", res)
					assertInCode(t, k+") validateIntThingEnum(path, location string, value int32)", res)
					assertInCode(t, "m.validateIntThingEnum(\"\", \"body\", m)", res)
				}
			}
		}
	}
}

func TestEnum_FloatThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "FloatThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("float_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var floatThingEnum []interface{}", res)
					assertInCode(t, k+") validateFloatThingEnum(path, location string, value float32)", res)
					assertInCode(t, "m.validateFloatThingEnum(\"\", \"body\", m)", res)
				}
			}
		}
	}
}

func TestEnum_SliceThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SliceThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("slice_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var sliceThingEnum []interface{}", res)
					assertInCode(t, k+") validateSliceThingEnum(path, location string, value []string)", res)
					assertInCode(t, "m.validateSliceThingEnum(\"\", \"body\", m)", res)
				}
			}
		}
	}
}

func TestEnum_SliceAndItemsThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SliceAndItemsThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("slice_and_items_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var sliceAndItemsThingEnum []interface{}", res)
					assertInCode(t, k+") validateSliceAndItemsThingEnum(path, location string, value []string)", res)
					assertInCode(t, "m.validateSliceAndItemsThingEnum(\"\", \"body\", m)", res)
					assertInCode(t, "var sliceAndItemsThingItemsEnum []interface{}", res)
					assertInCode(t, k+") validateSliceAndItemsThingItemsEnum(path, location string, value string)", res)
					assertInCode(t, "m.validateSliceAndItemsThingItemsEnum(strconv.Itoa(i), \"body\", m[i])", res)
				}
			}
		}
	}
}

func TestEnum_SliceAndAdditionalItemsThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SliceAndAdditionalItemsThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("slice_and_additional_items_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var sliceAndAdditionalItemsThingEnum []interface{}", res)
					assertInCode(t, k+") validateSliceAndAdditionalItemsThingEnum(path, location string, value SliceAndAdditionalItemsThing)", res)
					//assertInCode(t, "m.validateSliceAndAdditionalItemsThingEnum(\"\", \"body\", m)", res)
					assertInCode(t, "var sliceAndAdditionalItemsThingTypeP0PropEnum []interface{}", res)
					assertInCode(t, k+") validateP0Enum(path, location string, value string)", res)
					assertInCode(t, "m.validateP0Enum(\"0\", \"body\", m.P0)", res)
					assertInCode(t, "var sliceAndAdditionalItemsThingItemsEnum []interface{}", res)
					assertInCode(t, k+") validateSliceAndAdditionalItemsThingItemsEnum(path, location string, value float32)", res)
					assertInCode(t, "m.validateSliceAndAdditionalItemsThingItemsEnum(strconv.Itoa(i+1), \"body\", *m.SliceAndAdditionalItemsThingItems[i])", res)
				}
			}
		}
	}
}

func TestEnum_MapThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "MapThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("map_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var mapThingEnum []interface{}", res)
					assertInCode(t, k+") validateMapThingEnum(path, location string, value map[string]string)", res)
					assertInCode(t, "m.validateMapThingEnum(\"\", \"body\", m)", res)
					assertInCode(t, "var mapThingValueEnum []interface{}", res)
					assertInCode(t, k+") validateMapThingValueEnum(path, location string, value string)", res)
					assertInCode(t, "m.validateMapThingValueEnum(k, \"body\", m[k])", res)
				} else {
					fmt.Println(buf.String())
				}
			}
		}
	}
}

func TestEnum_ObjectThing(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "ObjectThing"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("object_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var objectThingTypeNamePropEnum []interface{}", res)
					assertInCode(t, "var objectThingTypeFlowerPropEnum []interface{}", res)
					assertInCode(t, "var objectThingTypeLionsPropEnum []interface{}", res)
					assertInCode(t, "var objectThingTypeFlourPropEnum []interface{}", res)
					assertInCode(t, "var objectThingTypeWolvesPropEnum []interface{}", res)
					assertInCode(t, "var objectThingWolvesValueEnum []interface{}", res)
					assertInCode(t, "var objectThingCatsItemsEnum []interface{}", res)
					assertInCode(t, "var objectThingLionsTuple0TypeP0PropEnum []interface{}", res)
					assertInCode(t, "var objectThingLionsTuple0TypeP1PropEnum []interface{}", res)
					assertInCode(t, "var objectThingLionsTuple0ItemsEnum []interface{}", res)
					assertInCode(t, k+") validateNameEnum(path, location string, value string)", res)
					assertInCode(t, k+") validateFlowerEnum(path, location string, value int32)", res)
					assertInCode(t, k+") validateFlourEnum(path, location string, value float32)", res)
					assertInCode(t, k+") validateLionsEnum(path, location string, value float64)", res)
					assertInCode(t, k+") validateWolvesEnum(path, location string, value map[string]string)", res)
					assertInCode(t, k+") validateWolvesValueEnum(path, location string, value string)", res)
					assertInCode(t, k+") validateCatsItemsEnum(path, location string, value string)", res)
					assertInCode(t, k+"LionsTuple0) validateObjectThingLionsTuple0ItemsEnum(path, location string, value float64)", res)
					assertInCode(t, k+") validateCats(", res)
					assertInCode(t, "m.validateNameEnum(\"name\", \"body\", m.Name)", res)
					assertInCode(t, "m.validateFlowerEnum(\"flower\", \"body\", *m.Flower)", res)
					assertInCode(t, "m.validateFlourEnum(\"flour\", \"body\", *m.Flour)", res)
					assertInCode(t, "m.validateWolvesEnum(\"wolves\", \"body\", m.Wolves)", res)
					assertInCode(t, "m.validateWolvesValueEnum(\"wolves\"+\".\"+k, \"body\", m.Wolves[k])", res)
					assertInCode(t, "m.validateCatsItemsEnum(\"cats\"+\".\"+strconv.Itoa(i), \"body\", m.Cats[i])", res)
					assertInCode(t, "m.validateP1Enum(\"P1\", \"body\", m.P1)", res)
					assertInCode(t, "m.validateP0Enum(\"P0\", \"body\", m.P0)", res)
					assertInCode(t, "m.validateObjectThingLionsTuple0ItemsEnum(strconv.Itoa(i), \"body\", *m.ObjectThingLionsTuple0Items[i])", res)
				}
			}
		}
	}
}

func TestEnum_ComputeInstance(t *testing.T) {
	// ensure that the enum validation for the anonymous object under the delegate property
	// is rendered.
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "ComputeInstance"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("object_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "Region string `json:\"region,omitempty\"`", res)
					assertInCode(t, "var computeInstanceTypeRegionPropEnum []interface{}", res)
					assertInCode(t, "m.validateRegionEnum(\"region\", \"body\", m.Region)", res)
				}
			}
		}
	}
}

func TestEnum_NewPrototype(t *testing.T) {
	// ensure that the enum validation for the anonymous object under the delegate property
	// is rendered.
	specDoc, err := spec.Load("../fixtures/codegen/todolist.enums.yml")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "NewPrototype"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("object_thing.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "ActivatingUser *NewPrototypeActivatingUser `json:\"activating_user,omitempty\"`", res)
					assertInCode(t, "Delegate *NewPrototypeDelegate `json:\"delegate,omitempty\"`", res)
					assertInCode(t, "Role string `json:\"role,omitempty\"`", res)
					assertInCode(t, "var newPrototypeTypeRolePropEnum []interface{}", res)
					assertInCode(t, "var newPrototypeDelegateTypeKindPropEnum []interface{}", res)
					assertInCode(t, "m.validateDelegate(formats)", res)
					assertInCode(t, "m.validateRole(formats)", res)
					assertInCode(t, "m.validateActivatingUser(formats)", res)
					assertInCode(t, "m.Delegate.Validate(formats)", res)
					assertInCode(t, "m.ActivatingUser.Validate(formats)", res)
				}
			}
		}
	}
}

func TestEnum_Issue265(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/sodabooth.json")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SodaBrand"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("soda_brand.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assert.Equal(t, 1, strings.Count(res, "m.validateSodaBrandEnum"))
				}
			}
		}
	}
}

func TestEnum_Issue325(t *testing.T) {
	specDoc, err := spec.Load("../fixtures/codegen/sodabooths.json")
	if assert.NoError(t, err) {
		definitions := specDoc.Spec().Definitions
		k := "SodaBrand"
		schema := definitions[k]
		genModel, err := makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("soda_brand.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var sodaBrandEnum []interface{}", res)
					assertInCode(t, "err := validate.Enum(path, location, value, sodaBrandEnum)", res)
					assert.Equal(t, 1, strings.Count(res, "m.validateSodaBrandEnum"))
				}
			}
		}

		k = "Soda"
		schema = definitions[k]
		genModel, err = makeGenDefinition(k, "models", schema, specDoc)
		if assert.NoError(t, err) {
			buf := bytes.NewBuffer(nil)
			err := modelTemplate.Execute(buf, genModel)
			if assert.NoError(t, err) {
				ff, err := formatGoFile("soda.go", buf.Bytes())
				if assert.NoError(t, err) {
					res := string(ff)
					assertInCode(t, "var sodaTypeBrandPropEnum []interface{}", res)
					assertInCode(t, "err := validate.Enum(path, location, value, sodaTypeBrandPropEnum)", res)
					assert.Equal(t, 1, strings.Count(res, "m.validateBrandEnum"))
				}
			}
		}
	}
}
