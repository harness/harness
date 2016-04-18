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
	"fmt"
	"testing"

	"github.com/go-swagger/go-swagger/errors"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/strfmt"
	"github.com/stretchr/testify/assert"
)

type email struct {
	Address string
}

type paramFactory func(string) *spec.Parameter

var paramFactories = []paramFactory{
	spec.QueryParam,
	spec.HeaderParam,
	spec.PathParam,
	spec.FormDataParam,
}

var stringItems = new(spec.Items)

func init() {
	stringItems.Type = "string"
}

func requiredError(param *spec.Parameter) *errors.Validation {
	return errors.Required(param.Name, param.In)
}

func maxErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.ExceedsMaximum(path, in, *items.Maximum, items.ExclusiveMaximum)
}

func minErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.ExceedsMinimum(path, in, *items.Minimum, items.ExclusiveMinimum)
}

func multipleOfErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.NotMultipleOf(path, in, *items.MultipleOf)
}

func requiredErrorItems(path, in string) *errors.Validation {
	return errors.Required(path, in)
}

func maxLengthErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.TooLong(path, in, *items.MaxLength)
}

func minLengthErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.TooShort(path, in, *items.MinLength)
}

func patternFailItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.FailedPattern(path, in, items.Pattern)
}

func enumFailItems(path, in string, items *spec.Items, data interface{}) *errors.Validation {
	return errors.EnumFail(path, in, data, items.Enum)
}

func minItemsErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.TooFewItems(path, in, *items.MinItems)
}

func maxItemsErrorItems(path, in string, items *spec.Items) *errors.Validation {
	return errors.TooManyItems(path, in, *items.MaxItems)
}

func duplicatesErrorItems(path, in string) *errors.Validation {
	return errors.DuplicateItems(path, in)
}

func TestNumberItemsValidation(t *testing.T) {

	values := [][]interface{}{
		[]interface{}{23, 49, 56, 21, 14, 35, 28, 7, 42},
		[]interface{}{uint(23), uint(49), uint(56), uint(21), uint(14), uint(35), uint(28), uint(7), uint(42)},
		[]interface{}{float64(23), float64(49), float64(56), float64(21), float64(14), float64(35), float64(28), float64(7), float64(42)},
	}

	for i, v := range values {
		items := spec.NewItems()
		items.WithMaximum(makeFloat(v[1]), false)
		items.WithMinimum(makeFloat(v[3]), false)
		items.WithMultipleOf(makeFloat(v[7]))
		items.WithEnum(v[3], v[6], v[8], v[1])
		items.Typed("integer", "int32")
		parent := spec.QueryParam("factors").CollectionOf(items, "")
		path := fmt.Sprintf("factors.%d", i)
		validator := newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)

		// MultipleOf
		err := validator.Validate(i, v[0])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, multipleOfErrorItems(path, validator.in, items), err.Errors[0].Error())

		// Maximum
		err = validator.Validate(i, v[1])
		assert.True(t, err == nil || err.IsValid())
		err = validator.Validate(i, v[2])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, maxErrorItems(path, validator.in, items), err.Errors[0].Error())

		// ExclusiveMaximum
		items.ExclusiveMaximum = true
		// requires a new items validator because this is set a creation time
		validator = newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)
		err = validator.Validate(i, v[1])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, maxErrorItems(path, validator.in, items), err.Errors[0].Error())

		// Minimum
		err = validator.Validate(i, v[3])
		assert.True(t, err == nil || err.IsValid())
		err = validator.Validate(i, v[4])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, minErrorItems(path, validator.in, items), err.Errors[0].Error())

		// ExclusiveMinimum
		items.ExclusiveMinimum = true
		// requires a new items validator because this is set a creation time
		validator = newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)
		err = validator.Validate(i, v[3])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, minErrorItems(path, validator.in, items), err.Errors[0].Error())

		// Enum
		err = validator.Validate(i, v[5])
		assert.True(t, err.HasErrors())
		assert.EqualError(t, enumFailItems(path, validator.in, items, v[5]), err.Errors[0].Error())

		// Valid passes
		err = validator.Validate(i, v[6])
		assert.True(t, err == nil || err.IsValid())
	}

}

func TestStringItemsValidation(t *testing.T) {
	items := spec.NewItems().WithMinLength(3).WithMaxLength(5).WithPattern(`^[a-z]+$`).Typed("string", "")
	items.WithEnum("aaa", "bbb", "ccc")
	parent := spec.QueryParam("tags").CollectionOf(items, "")
	path := parent.Name + ".1"
	validator := newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)

	// required
	err := validator.Validate(1, "")
	assert.True(t, err.HasErrors())
	assert.EqualError(t, minLengthErrorItems(path, validator.in, items), err.Errors[0].Error())

	// MaxLength
	err = validator.Validate(1, "abcdef")
	assert.True(t, err.HasErrors())
	assert.EqualError(t, maxLengthErrorItems(path, validator.in, items), err.Errors[0].Error())

	// MinLength
	err = validator.Validate(1, "a")
	assert.True(t, err.HasErrors())
	assert.EqualError(t, minLengthErrorItems(path, validator.in, items), err.Errors[0].Error())

	// Pattern
	err = validator.Validate(1, "a394")
	assert.True(t, err.HasErrors())
	assert.EqualError(t, patternFailItems(path, validator.in, items), err.Errors[0].Error())

	// Enum
	err = validator.Validate(1, "abcde")
	assert.True(t, err.HasErrors())
	assert.EqualError(t, enumFailItems(path, validator.in, items, "abcde"), err.Errors[0].Error())

	// Valid passes
	err = validator.Validate(1, "bbb")
	assert.True(t, err == nil || err.IsValid())
}

func TestArrayItemsValidation(t *testing.T) {
	items := spec.NewItems().CollectionOf(stringItems, "").WithMinItems(1).WithMaxItems(5).UniqueValues()
	items.WithEnum("aaa", "bbb", "ccc")
	parent := spec.QueryParam("tags").CollectionOf(items, "")
	path := parent.Name + ".1"
	validator := newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)

	// MinItems
	err := validator.Validate(1, []string{})
	assert.True(t, err.HasErrors())
	assert.EqualError(t, minItemsErrorItems(path, validator.in, items), err.Errors[0].Error())
	// MaxItems
	err = validator.Validate(1, []string{"a", "b", "c", "d", "e", "f"})
	assert.True(t, err.HasErrors())
	assert.EqualError(t, maxItemsErrorItems(path, validator.in, items), err.Errors[0].Error())
	// UniqueItems
	err = validator.Validate(1, []string{"a", "a"})
	assert.True(t, err.HasErrors())
	assert.EqualError(t, duplicatesErrorItems(path, validator.in), err.Errors[0].Error())

	// Enum
	err = validator.Validate(1, []string{"a", "b", "c"})
	assert.True(t, err.HasErrors())
	assert.EqualError(t, enumFailItems(path, validator.in, items, []string{"a", "b", "c"}), err.Errors[0].Error())

	// Items
	strItems := spec.NewItems().WithMinLength(3).WithMaxLength(5).WithPattern(`^[a-z]+$`).Typed("string", "")
	items = spec.NewItems().CollectionOf(strItems, "").WithMinItems(1).WithMaxItems(5).UniqueValues()
	validator = newItemsValidator(parent.Name, parent.In, items, parent, strfmt.Default)

	err = validator.Validate(1, []string{"aa", "bbb", "ccc"})
	assert.True(t, err.HasErrors())
	assert.EqualError(t, minLengthErrorItems(path+".0", parent.In, strItems), err.Errors[0].Error())
}
