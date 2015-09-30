package validator_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"gopkg.in/bluesuncorp/go-validate-yourself.v4"
	. "gopkg.in/check.v1"
)

type I interface {
	Foo() string
}

type Impl struct {
	F string `validate:"len=3"`
}

func (i *Impl) Foo() string {
	return i.F
}

type SubTest struct {
	Test string `validate:"required"`
}

type TestInterface struct {
	Iface I
}

type TestString struct {
	BlankTag  string `validate:""`
	Required  string `validate:"required"`
	Len       string `validate:"len=10"`
	Min       string `validate:"min=1"`
	Max       string `validate:"max=10"`
	MinMax    string `validate:"min=1,max=10"`
	Lt        string `validate:"lt=10"`
	Lte       string `validate:"lte=10"`
	Gt        string `validate:"gt=10"`
	Gte       string `validate:"gte=10"`
	OmitEmpty string `validate:"omitempty,min=1,max=10"`
	Sub       *SubTest
	SubIgnore *SubTest `validate:"-"`
	Anonymous struct {
		A string `validate:"required"`
	}
	Iface I
}

type TestInt32 struct {
	Required  int `validate:"required"`
	Len       int `validate:"len=10"`
	Min       int `validate:"min=1"`
	Max       int `validate:"max=10"`
	MinMax    int `validate:"min=1,max=10"`
	Lt        int `validate:"lt=10"`
	Lte       int `validate:"lte=10"`
	Gt        int `validate:"gt=10"`
	Gte       int `validate:"gte=10"`
	OmitEmpty int `validate:"omitempty,min=1,max=10"`
}

type TestUint64 struct {
	Required  uint64 `validate:"required"`
	Len       uint64 `validate:"len=10"`
	Min       uint64 `validate:"min=1"`
	Max       uint64 `validate:"max=10"`
	MinMax    uint64 `validate:"min=1,max=10"`
	OmitEmpty uint64 `validate:"omitempty,min=1,max=10"`
}

type TestFloat64 struct {
	Required  float64 `validate:"required"`
	Len       float64 `validate:"len=10"`
	Min       float64 `validate:"min=1"`
	Max       float64 `validate:"max=10"`
	MinMax    float64 `validate:"min=1,max=10"`
	Lte       float64 `validate:"lte=10"`
	OmitEmpty float64 `validate:"omitempty,min=1,max=10"`
}

type TestSlice struct {
	Required  []int `validate:"required"`
	Len       []int `validate:"len=10"`
	Min       []int `validate:"min=1"`
	Max       []int `validate:"max=10"`
	MinMax    []int `validate:"min=1,max=10"`
	OmitEmpty []int `validate:"omitempty,min=1,max=10"`
}

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

var myValidator = validator.NewValidator("validate", validator.BakedInValidators)

func AssetStruct(s *validator.StructValidationErrors, structFieldName string, expectedStructName string, c *C) *validator.StructValidationErrors {

	val, ok := s.StructErrors[structFieldName]
	c.Assert(ok, Equals, true)
	c.Assert(val, NotNil)
	c.Assert(val.Struct, Equals, expectedStructName)

	return val
}

func AssertFieldError(s *validator.StructValidationErrors, field string, expectedTag string, c *C) {

	val, ok := s.Errors[field]
	c.Assert(ok, Equals, true)
	c.Assert(val, NotNil)
	c.Assert(val.Field, Equals, field)
	c.Assert(val.ErrorTag, Equals, expectedTag)
}

func AssertMapFieldError(s map[string]*validator.FieldValidationError, field string, expectedTag string, c *C) {

	val, ok := s[field]
	c.Assert(ok, Equals, true)
	c.Assert(val, NotNil)
	c.Assert(val.Field, Equals, field)
	c.Assert(val.ErrorTag, Equals, expectedTag)
}

func newValidatorFunc(val interface{}, current interface{}, field interface{}, param string) bool {

	return true
}

func isEqualFunc(val interface{}, current interface{}, field interface{}, param string) bool {

	return current.(string) == field.(string)
}

func (ms *MySuite) TestStructOnlyValidation(c *C) {

	type Inner struct {
		Test string `validate:"len=5"`
	}

	type Outer struct {
		InnerStruct *Inner `validate:"required,structonly"`
	}

	outer := &Outer{
		InnerStruct: nil,
	}

	errs := myValidator.ValidateStruct(outer).Flatten()
	c.Assert(errs, NotNil)

	inner := &Inner{
		Test: "1234",
	}

	outer = &Outer{
		InnerStruct: inner,
	}

	errs = myValidator.ValidateStruct(outer).Flatten()
	c.Assert(errs, IsNil)
}

func (ms *MySuite) TestGtField(c *C) {

	type TimeTest struct {
		Start *time.Time `validate:"required,gt"`
		End   *time.Time `validate:"required,gt,gtfield=Start"`
	}

	now := time.Now()
	start := now.Add(time.Hour * 24)
	end := start.Add(time.Hour * 24)

	timeTest := &TimeTest{
		Start: &start,
		End:   &end,
	}

	errs := myValidator.ValidateStruct(timeTest)
	c.Assert(errs, IsNil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := myValidator.ValidateStruct(timeTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "End", "gtfield", c)

	err3 := myValidator.ValidateFieldByTagAndValue(&start, &end, "gtfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(&end, &start, "gtfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtfield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,gtfield=Val1"`
	}

	intTest := &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(intTest)
	c.Assert(errs, IsNil)

	intTest = &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(intTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(int(1), int(5), "gtfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(int(5), int(1), "gtfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtfield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,gtfield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(uIntTest)
	c.Assert(errs, IsNil)

	uIntTest = &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(uIntTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(1), uint(5), "gtfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(5), uint(1), "gtfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtfield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,gtfield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(floatTest)
	c.Assert(errs, IsNil)

	floatTest = &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(floatTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(1), float32(5), "gtfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(5), float32(1), "gtfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtfield")

	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(nil, 1, "gtfield") }, PanicMatches, "struct not passed for cross validation")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(5, "T", "gtfield") }, PanicMatches, "Bad field type string")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(5, start, "gtfield") }, PanicMatches, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,gtfield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &start,
		End:   &end,
	}

	c.Assert(func() { myValidator.ValidateStruct(timeTest2) }, PanicMatches, "Field \"NonExistantField\" not found in struct")
}

func (ms *MySuite) TestLtField(c *C) {

	type TimeTest struct {
		Start *time.Time `validate:"required,lt,ltfield=End"`
		End   *time.Time `validate:"required,lt"`
	}

	now := time.Now()
	start := now.Add(time.Hour * 24 * -1 * 2)
	end := start.Add(time.Hour * 24)

	timeTest := &TimeTest{
		Start: &start,
		End:   &end,
	}

	errs := myValidator.ValidateStruct(timeTest)
	c.Assert(errs, IsNil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := myValidator.ValidateStruct(timeTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Start", "ltfield", c)

	err3 := myValidator.ValidateFieldByTagAndValue(&end, &start, "ltfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(&start, &end, "ltfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltfield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,ltfield=Val1"`
	}

	intTest := &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(intTest)
	c.Assert(errs, IsNil)

	intTest = &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(intTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(int(5), int(1), "ltfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(int(1), int(5), "ltfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltfield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,ltfield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(uIntTest)
	c.Assert(errs, IsNil)

	uIntTest = &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(uIntTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(5), uint(1), "ltfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(1), uint(5), "ltfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltfield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,ltfield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(floatTest)
	c.Assert(errs, IsNil)

	floatTest = &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(floatTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltfield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(5), float32(1), "ltfield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(1), float32(5), "ltfield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltfield")

	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(nil, 5, "ltfield") }, PanicMatches, "struct not passed for cross validation")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(1, "T", "ltfield") }, PanicMatches, "Bad field type string")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(1, end, "ltfield") }, PanicMatches, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,ltfield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &end,
		End:   &start,
	}

	c.Assert(func() { myValidator.ValidateStruct(timeTest2) }, PanicMatches, "Field \"NonExistantField\" not found in struct")
}

func (ms *MySuite) TestLteField(c *C) {

	type TimeTest struct {
		Start *time.Time `validate:"required,lte,ltefield=End"`
		End   *time.Time `validate:"required,lte"`
	}

	now := time.Now()
	start := now.Add(time.Hour * 24 * -1 * 2)
	end := start.Add(time.Hour * 24)

	timeTest := &TimeTest{
		Start: &start,
		End:   &end,
	}

	errs := myValidator.ValidateStruct(timeTest)
	c.Assert(errs, IsNil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := myValidator.ValidateStruct(timeTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Start", "ltefield", c)

	err3 := myValidator.ValidateFieldByTagAndValue(&end, &start, "ltefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(&start, &end, "ltefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltefield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,ltefield=Val1"`
	}

	intTest := &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(intTest)
	c.Assert(errs, IsNil)

	intTest = &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(intTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(int(5), int(1), "ltefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(int(1), int(5), "ltefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltefield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,ltefield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(uIntTest)
	c.Assert(errs, IsNil)

	uIntTest = &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(uIntTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(5), uint(1), "ltefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(1), uint(5), "ltefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltefield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,ltefield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs = myValidator.ValidateStruct(floatTest)
	c.Assert(errs, IsNil)

	floatTest = &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = myValidator.ValidateStruct(floatTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "ltefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(5), float32(1), "ltefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(1), float32(5), "ltefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "ltefield")

	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(nil, 5, "ltefield") }, PanicMatches, "struct not passed for cross validation")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(1, "T", "ltefield") }, PanicMatches, "Bad field type string")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(1, end, "ltefield") }, PanicMatches, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,ltefield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &end,
		End:   &start,
	}

	c.Assert(func() { myValidator.ValidateStruct(timeTest2) }, PanicMatches, "Field \"NonExistantField\" not found in struct")
}

func (ms *MySuite) TestGteField(c *C) {

	type TimeTest struct {
		Start *time.Time `validate:"required,gte"`
		End   *time.Time `validate:"required,gte,gtefield=Start"`
	}

	now := time.Now()
	start := now.Add(time.Hour * 24)
	end := start.Add(time.Hour * 24)

	timeTest := &TimeTest{
		Start: &start,
		End:   &end,
	}

	errs := myValidator.ValidateStruct(timeTest)
	c.Assert(errs, IsNil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := myValidator.ValidateStruct(timeTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "End", "gtefield", c)

	err3 := myValidator.ValidateFieldByTagAndValue(&start, &end, "gtefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(&end, &start, "gtefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtefield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,gtefield=Val1"`
	}

	intTest := &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(intTest)
	c.Assert(errs, IsNil)

	intTest = &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(intTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(int(1), int(5), "gtefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(int(5), int(1), "gtefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtefield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,gtefield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(uIntTest)
	c.Assert(errs, IsNil)

	uIntTest = &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(uIntTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(1), uint(5), "gtefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(uint(5), uint(1), "gtefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtefield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,gtefield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs = myValidator.ValidateStruct(floatTest)
	c.Assert(errs, IsNil)

	floatTest = &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = myValidator.ValidateStruct(floatTest).Flatten()
	c.Assert(errs2, NotNil)
	AssertMapFieldError(errs2, "Val2", "gtefield", c)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(1), float32(5), "gtefield")
	c.Assert(err3, IsNil)

	err3 = myValidator.ValidateFieldByTagAndValue(float32(5), float32(1), "gtefield")
	c.Assert(err3, NotNil)
	c.Assert(err3.ErrorTag, Equals, "gtefield")

	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(nil, 1, "gtefield") }, PanicMatches, "struct not passed for cross validation")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(5, "T", "gtefield") }, PanicMatches, "Bad field type string")
	c.Assert(func() { myValidator.ValidateFieldByTagAndValue(5, start, "gtefield") }, PanicMatches, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,gtefield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &start,
		End:   &end,
	}

	c.Assert(func() { myValidator.ValidateStruct(timeTest2) }, PanicMatches, "Field \"NonExistantField\" not found in struct")
}

func (ms *MySuite) TestValidateByTagAndValue(c *C) {

	val := "test"
	field := "test"
	err := myValidator.ValidateFieldByTagAndValue(val, field, "required")
	c.Assert(err, IsNil)

	myValidator.AddFunction("isequaltestfunc", isEqualFunc)

	err = myValidator.ValidateFieldByTagAndValue(val, field, "isequaltestfunc")
	c.Assert(err, IsNil)

	val = "unequal"

	err = myValidator.ValidateFieldByTagAndValue(val, field, "isequaltestfunc")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "isequaltestfunc")
}

func (ms *MySuite) TestAddFunctions(c *C) {

	validate := validator.NewValidator("validateme", validator.BakedInValidators)

	err := validate.AddFunction("new", newValidatorFunc)
	c.Assert(err, IsNil)

	err = validate.AddFunction("", newValidatorFunc)
	c.Assert(err, NotNil)

	validate.AddFunction("new", nil)
	c.Assert(err, NotNil)

	err = validate.AddFunction("new", newValidatorFunc)
	c.Assert(err, IsNil)
}

func (ms *MySuite) TestChangeTag(c *C) {

	validate := validator.NewValidator("validateme", validator.BakedInValidators)
	validate.SetTag("val")

	type Test struct {
		Name string `val:"len=4"`
	}
	s := &Test{
		Name: "TEST",
	}

	err := validate.ValidateStruct(s)
	c.Assert(err, IsNil)

	// validator.SetTag("v")
	// validator.SetTag("validate")
}

func (ms *MySuite) TestUnexposedStruct(c *C) {

	type Test struct {
		Name      string
		unexposed struct {
			A string `validate:"required"`
		}
	}

	s := &Test{
		Name: "TEST",
	}

	err := myValidator.ValidateStruct(s)
	c.Assert(err, IsNil)
}

func (ms *MySuite) TestBadParams(c *C) {

	i := 1
	err := myValidator.ValidateFieldByTag(i, "-")
	c.Assert(err, IsNil)

	c.Assert(func() { myValidator.ValidateFieldByTag(i, "len=a") }, PanicMatches, "strconv.ParseInt: parsing \"a\": invalid syntax")
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "len=a") }, PanicMatches, "strconv.ParseInt: parsing \"a\": invalid syntax")

	var ui uint = 1
	c.Assert(func() { myValidator.ValidateFieldByTag(ui, "len=a") }, PanicMatches, "strconv.ParseUint: parsing \"a\": invalid syntax")

	f := 1.23
	c.Assert(func() { myValidator.ValidateFieldByTag(f, "len=a") }, PanicMatches, "strconv.ParseFloat: parsing \"a\": invalid syntax")
}

func (ms *MySuite) TestLength(c *C) {

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "len") }, PanicMatches, "Bad field type bool")
}

func (ms *MySuite) TestIsGt(c *C) {

	myMap := map[string]string{}
	err := myValidator.ValidateFieldByTag(myMap, "gt=0")
	c.Assert(err, NotNil)

	f := 1.23
	err = myValidator.ValidateFieldByTag(f, "gt=5")
	c.Assert(err, NotNil)

	var ui uint = 5
	err = myValidator.ValidateFieldByTag(ui, "gt=10")
	c.Assert(err, NotNil)

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "gt") }, PanicMatches, "Bad field type bool")

	t := time.Now().UTC()
	t = t.Add(time.Hour * 24)

	err = myValidator.ValidateFieldByTag(t, "gt")
	c.Assert(err, IsNil)

	t2 := time.Now().UTC()

	err = myValidator.ValidateFieldByTag(t2, "gt")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "gt")

	type Test struct {
		Now *time.Time `validate:"gt"`
	}
	s := &Test{
		Now: &t,
	}

	errs := myValidator.ValidateStruct(s)
	c.Assert(errs, IsNil)

	s = &Test{
		Now: &t2,
	}

	errs = myValidator.ValidateStruct(s)
	c.Assert(errs, NotNil)
}

func (ms *MySuite) TestIsGte(c *C) {

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "gte") }, PanicMatches, "Bad field type bool")

	t := time.Now().UTC()
	t = t.Add(time.Hour * 24)

	err := myValidator.ValidateFieldByTag(t, "gte")
	c.Assert(err, IsNil)

	t2 := time.Now().UTC()

	err = myValidator.ValidateFieldByTag(t2, "gte")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "gte")
	c.Assert(err.Type, Equals, reflect.TypeOf(time.Time{}))

	type Test struct {
		Now *time.Time `validate:"gte"`
	}
	s := &Test{
		Now: &t,
	}

	errs := myValidator.ValidateStruct(s)
	c.Assert(errs, IsNil)

	s = &Test{
		Now: &t2,
	}

	errs = myValidator.ValidateStruct(s)
	c.Assert(errs, NotNil)
}

func (ms *MySuite) TestIsLt(c *C) {

	myMap := map[string]string{}
	err := myValidator.ValidateFieldByTag(myMap, "lt=0")
	c.Assert(err, NotNil)

	f := 1.23
	err = myValidator.ValidateFieldByTag(f, "lt=0")
	c.Assert(err, NotNil)

	var ui uint = 5
	err = myValidator.ValidateFieldByTag(ui, "lt=0")
	c.Assert(err, NotNil)

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "lt") }, PanicMatches, "Bad field type bool")

	t := time.Now().UTC()

	err = myValidator.ValidateFieldByTag(t, "lt")
	c.Assert(err, IsNil)

	t2 := time.Now().UTC()
	t2 = t2.Add(time.Hour * 24)

	err = myValidator.ValidateFieldByTag(t2, "lt")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "lt")

	type Test struct {
		Now *time.Time `validate:"lt"`
	}

	s := &Test{
		Now: &t,
	}

	errs := myValidator.ValidateStruct(s)
	c.Assert(errs, IsNil)

	s = &Test{
		Now: &t2,
	}

	errs = myValidator.ValidateStruct(s)
	c.Assert(errs, NotNil)
}

func (ms *MySuite) TestIsLte(c *C) {

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "lte") }, PanicMatches, "Bad field type bool")

	t := time.Now().UTC()

	err := myValidator.ValidateFieldByTag(t, "lte")
	c.Assert(err, IsNil)

	t2 := time.Now().UTC()
	t2 = t2.Add(time.Hour * 24)

	err = myValidator.ValidateFieldByTag(t2, "lte")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "lte")

	type Test struct {
		Now *time.Time `validate:"lte"`
	}

	s := &Test{
		Now: &t,
	}

	errs := myValidator.ValidateStruct(s)
	c.Assert(errs, IsNil)

	s = &Test{
		Now: &t2,
	}

	errs = myValidator.ValidateStruct(s)
	c.Assert(errs, NotNil)
}

func (ms *MySuite) TestUrl(c *C) {

	var tests = []struct {
		param    string
		expected bool
	}{
		{"http://foo.bar#com", true},
		{"http://foobar.com", true},
		{"https://foobar.com", true},
		{"foobar.com", false},
		{"http://foobar.coffee/", true},
		{"http://foobar.中文网/", true},
		{"http://foobar.org/", true},
		{"http://foobar.org:8080/", true},
		{"ftp://foobar.ru/", true},
		{"http://user:pass@www.foobar.com/", true},
		{"http://127.0.0.1/", true},
		{"http://duckduckgo.com/?q=%2F", true},
		{"http://localhost:3000/", true},
		{"http://foobar.com/?foo=bar#baz=qux", true},
		{"http://foobar.com?foo=bar", true},
		{"http://www.xn--froschgrn-x9a.net/", true},
		{"", false},
		{"xyz://foobar.com", true},
		{"invalid.", false},
		{".com", false},
		{"rtmp://foobar.com", true},
		{"http://www.foo_bar.com/", true},
		{"http://localhost:3000/", true},
		{"http://foobar.com#baz=qux", true},
		{"http://foobar.com/t$-_.+!*\\'(),", true},
		{"http://www.foobar.com/~foobar", true},
		{"http://www.-foobar.com/", true},
		{"http://www.foo---bar.com/", true},
		{"mailto:someone@example.com", true},
		{"irc://irc.server.org/channel", true},
		{"irc://#channel@network", true},
		{"/abs/test/dir", false},
		{"./rel/test/dir", false},
	}
	for _, test := range tests {

		err := myValidator.ValidateFieldByTag(test.param, "url")

		if test.expected == true {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err, NotNil)
			c.Assert(err.ErrorTag, Equals, "url")
		}
	}

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "url") }, PanicMatches, "Bad field type int")
}

func (ms *MySuite) TestUri(c *C) {

	var tests = []struct {
		param    string
		expected bool
	}{
		{"http://foo.bar#com", true},
		{"http://foobar.com", true},
		{"https://foobar.com", true},
		{"foobar.com", false},
		{"http://foobar.coffee/", true},
		{"http://foobar.中文网/", true},
		{"http://foobar.org/", true},
		{"http://foobar.org:8080/", true},
		{"ftp://foobar.ru/", true},
		{"http://user:pass@www.foobar.com/", true},
		{"http://127.0.0.1/", true},
		{"http://duckduckgo.com/?q=%2F", true},
		{"http://localhost:3000/", true},
		{"http://foobar.com/?foo=bar#baz=qux", true},
		{"http://foobar.com?foo=bar", true},
		{"http://www.xn--froschgrn-x9a.net/", true},
		{"", false},
		{"xyz://foobar.com", true},
		{"invalid.", false},
		{".com", false},
		{"rtmp://foobar.com", true},
		{"http://www.foo_bar.com/", true},
		{"http://localhost:3000/", true},
		{"http://foobar.com#baz=qux", true},
		{"http://foobar.com/t$-_.+!*\\'(),", true},
		{"http://www.foobar.com/~foobar", true},
		{"http://www.-foobar.com/", true},
		{"http://www.foo---bar.com/", true},
		{"mailto:someone@example.com", true},
		{"irc://irc.server.org/channel", true},
		{"irc://#channel@network", true},
		{"/abs/test/dir", true},
		{"./rel/test/dir", false},
	}
	for _, test := range tests {

		err := myValidator.ValidateFieldByTag(test.param, "uri")

		if test.expected == true {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err, NotNil)
			c.Assert(err.ErrorTag, Equals, "uri")
		}
	}

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "uri") }, PanicMatches, "Bad field type int")
}

func (ms *MySuite) TestOrTag(c *C) {
	s := "rgba(0,31,255,0.5)"
	err := myValidator.ValidateFieldByTag(s, "rgb|rgba")
	c.Assert(err, IsNil)

	s = "rgba(0,31,255,0.5)"
	err = myValidator.ValidateFieldByTag(s, "rgb|rgba|len=18")
	c.Assert(err, IsNil)

	s = "this ain't right"
	err = myValidator.ValidateFieldByTag(s, "rgb|rgba")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgb|rgba")

	s = "this ain't right"
	err = myValidator.ValidateFieldByTag(s, "rgb|rgba|len=10")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgb|rgba|len")

	s = "this is right"
	err = myValidator.ValidateFieldByTag(s, "rgb|rgba|len=13")
	c.Assert(err, IsNil)

	s = ""
	err = myValidator.ValidateFieldByTag(s, "omitempty,rgb|rgba")
	c.Assert(err, IsNil)
}

func (ms *MySuite) TestHsla(c *C) {

	s := "hsla(360,100%,100%,1)"
	err := myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, IsNil)

	s = "hsla(360,100%,100%,0.5)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, IsNil)

	s = "hsla(0,0%,0%, 0)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, IsNil)

	s = "hsl(361,100%,50%,1)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsla")

	s = "hsl(361,100%,50%)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsla")

	s = "hsla(361,100%,50%)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsla")

	s = "hsla(360,101%,50%)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsla")

	s = "hsla(360,100%,101%)"
	err = myValidator.ValidateFieldByTag(s, "hsla")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsla")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "hsla") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestHsl(c *C) {

	s := "hsl(360,100%,50%)"
	err := myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, IsNil)

	s = "hsl(0,0%,0%)"
	err = myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, IsNil)

	s = "hsl(361,100%,50%)"
	err = myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsl")

	s = "hsl(361,101%,50%)"
	err = myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsl")

	s = "hsl(361,100%,101%)"
	err = myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsl")

	s = "hsl(-10,100%,100%)"
	err = myValidator.ValidateFieldByTag(s, "hsl")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hsl")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "hsl") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestRgba(c *C) {

	s := "rgba(0,31,255,0.5)"
	err := myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, IsNil)

	s = "rgba(0,31,255,0.12)"
	err = myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, IsNil)

	s = "rgba( 0,  31, 255, 0.5)"
	err = myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, IsNil)

	s = "rgb(0,  31, 255)"
	err = myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgba")

	s = "rgb(1,349,275,0.5)"
	err = myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgba")

	s = "rgb(01,31,255,0.5)"
	err = myValidator.ValidateFieldByTag(s, "rgba")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgba")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "rgba") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestRgb(c *C) {

	s := "rgb(0,31,255)"
	err := myValidator.ValidateFieldByTag(s, "rgb")
	c.Assert(err, IsNil)

	s = "rgb(0,  31, 255)"
	err = myValidator.ValidateFieldByTag(s, "rgb")
	c.Assert(err, IsNil)

	s = "rgb(1,349,275)"
	err = myValidator.ValidateFieldByTag(s, "rgb")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgb")

	s = "rgb(01,31,255)"
	err = myValidator.ValidateFieldByTag(s, "rgb")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgb")

	s = "rgba(0,31,255)"
	err = myValidator.ValidateFieldByTag(s, "rgb")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "rgb")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "rgb") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestEmail(c *C) {

	s := "test@mail.com"
	err := myValidator.ValidateFieldByTag(s, "email")
	c.Assert(err, IsNil)

	s = ""
	err = myValidator.ValidateFieldByTag(s, "email")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "email")

	s = "test@email"
	err = myValidator.ValidateFieldByTag(s, "email")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "email")

	s = "test@email."
	err = myValidator.ValidateFieldByTag(s, "email")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "email")

	s = "@email.com"
	err = myValidator.ValidateFieldByTag(s, "email")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "email")

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "email") }, PanicMatches, "interface conversion: interface is bool, not string")
}

func (ms *MySuite) TestHexColor(c *C) {

	s := "#fff"
	err := myValidator.ValidateFieldByTag(s, "hexcolor")
	c.Assert(err, IsNil)

	s = "#c2c2c2"
	err = myValidator.ValidateFieldByTag(s, "hexcolor")
	c.Assert(err, IsNil)

	s = "fff"
	err = myValidator.ValidateFieldByTag(s, "hexcolor")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hexcolor")

	s = "fffFF"
	err = myValidator.ValidateFieldByTag(s, "hexcolor")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hexcolor")

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "hexcolor") }, PanicMatches, "interface conversion: interface is bool, not string")
}

func (ms *MySuite) TestHexadecimal(c *C) {

	s := "ff0044"
	err := myValidator.ValidateFieldByTag(s, "hexadecimal")
	c.Assert(err, IsNil)

	s = "abcdefg"
	err = myValidator.ValidateFieldByTag(s, "hexadecimal")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "hexadecimal")

	i := true
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "hexadecimal") }, PanicMatches, "interface conversion: interface is bool, not string")
}

func (ms *MySuite) TestNumber(c *C) {

	s := "1"
	err := myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, IsNil)

	s = "+1"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "-1"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "1.12"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "+1.12"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "-1.12"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "1."
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	s = "1.o"
	err = myValidator.ValidateFieldByTag(s, "number")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "number")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "number") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestNumeric(c *C) {

	s := "1"
	err := myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "+1"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "-1"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "1.12"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "+1.12"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "-1.12"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, IsNil)

	s = "1."
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "numeric")

	s = "1.o"
	err = myValidator.ValidateFieldByTag(s, "numeric")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "numeric")

	i := 1
	c.Assert(func() { myValidator.ValidateFieldByTag(i, "numeric") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestAlphaNumeric(c *C) {

	s := "abcd123"
	err := myValidator.ValidateFieldByTag(s, "alphanum")
	c.Assert(err, IsNil)

	s = "abc!23"
	err = myValidator.ValidateFieldByTag(s, "alphanum")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "alphanum")

	c.Assert(func() { myValidator.ValidateFieldByTag(1, "alphanum") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestAlpha(c *C) {

	s := "abcd"
	err := myValidator.ValidateFieldByTag(s, "alpha")
	c.Assert(err, IsNil)

	s = "abc1"
	err = myValidator.ValidateFieldByTag(s, "alpha")
	c.Assert(err, NotNil)
	c.Assert(err.ErrorTag, Equals, "alpha")

	c.Assert(func() { myValidator.ValidateFieldByTag(1, "alpha") }, PanicMatches, "interface conversion: interface is int, not string")
}

func (ms *MySuite) TestFlattening(c *C) {

	tSuccess := &TestString{
		Required:  "Required",
		Len:       "length==10",
		Min:       "min=1",
		Max:       "1234567890",
		MinMax:    "12345",
		Lt:        "012345678",
		Lte:       "0123456789",
		Gt:        "01234567890",
		Gte:       "0123456789",
		OmitEmpty: "",
		Sub: &SubTest{
			Test: "1",
		},
		SubIgnore: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "1",
		},
		Iface: &Impl{
			F: "123",
		},
	}

	err1 := myValidator.ValidateStruct(tSuccess).Flatten()
	c.Assert(err1, IsNil)

	tFail := &TestString{
		Required:  "",
		Len:       "",
		Min:       "",
		Max:       "12345678901",
		MinMax:    "",
		OmitEmpty: "12345678901",
		Sub: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "",
		},
		Iface: &Impl{
			F: "12",
		},
	}

	err2 := myValidator.ValidateStruct(tFail).Flatten()

	// Assert Top Level
	c.Assert(err2, NotNil)

	// Assert Fields
	AssertMapFieldError(err2, "Len", "len", c)
	AssertMapFieldError(err2, "Gt", "gt", c)
	AssertMapFieldError(err2, "Gte", "gte", c)

	// Assert Struct Field
	AssertMapFieldError(err2, "Sub.Test", "required", c)

	// Assert Anonymous Struct Field
	AssertMapFieldError(err2, "Anonymous.A", "required", c)

	// Assert Interface Field
	AssertMapFieldError(err2, "Iface.F", "len", c)
}

func (ms *MySuite) TestStructStringValidation(c *C) {

	tSuccess := &TestString{
		Required:  "Required",
		Len:       "length==10",
		Min:       "min=1",
		Max:       "1234567890",
		MinMax:    "12345",
		Lt:        "012345678",
		Lte:       "0123456789",
		Gt:        "01234567890",
		Gte:       "0123456789",
		OmitEmpty: "",
		Sub: &SubTest{
			Test: "1",
		},
		SubIgnore: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "1",
		},
		Iface: &Impl{
			F: "123",
		},
	}

	err := myValidator.ValidateStruct(tSuccess)
	c.Assert(err, IsNil)

	tFail := &TestString{
		Required:  "",
		Len:       "",
		Min:       "",
		Max:       "12345678901",
		MinMax:    "",
		Lt:        "0123456789",
		Lte:       "01234567890",
		Gt:        "1",
		Gte:       "1",
		OmitEmpty: "12345678901",
		Sub: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "",
		},
		Iface: &Impl{
			F: "12",
		},
	}

	err = myValidator.ValidateStruct(tFail)

	// Assert Top Level
	c.Assert(err, NotNil)
	c.Assert(err.Struct, Equals, "TestString")
	c.Assert(len(err.Errors), Equals, 10)
	c.Assert(len(err.StructErrors), Equals, 3)

	// Assert Fields
	AssertFieldError(err, "Required", "required", c)
	AssertFieldError(err, "Len", "len", c)
	AssertFieldError(err, "Min", "min", c)
	AssertFieldError(err, "Max", "max", c)
	AssertFieldError(err, "MinMax", "min", c)
	AssertFieldError(err, "Gt", "gt", c)
	AssertFieldError(err, "Gte", "gte", c)
	AssertFieldError(err, "OmitEmpty", "max", c)

	// Assert Anonymous embedded struct
	AssetStruct(err, "Anonymous", "", c)

	// Assert SubTest embedded struct
	val := AssetStruct(err, "Sub", "SubTest", c)
	c.Assert(len(val.Errors), Equals, 1)
	c.Assert(len(val.StructErrors), Equals, 0)

	AssertFieldError(val, "Test", "required", c)

	errors := err.Error()
	c.Assert(errors, NotNil)
}

func (ms *MySuite) TestStructInt32Validation(c *C) {

	tSuccess := &TestInt32{
		Required:  1,
		Len:       10,
		Min:       1,
		Max:       10,
		MinMax:    5,
		Lt:        9,
		Lte:       10,
		Gt:        11,
		Gte:       10,
		OmitEmpty: 0,
	}

	err := myValidator.ValidateStruct(tSuccess)
	c.Assert(err, IsNil)

	tFail := &TestInt32{
		Required:  0,
		Len:       11,
		Min:       -1,
		Max:       11,
		MinMax:    -1,
		Lt:        10,
		Lte:       11,
		Gt:        10,
		Gte:       9,
		OmitEmpty: 11,
	}

	err = myValidator.ValidateStruct(tFail)

	// Assert Top Level
	c.Assert(err, NotNil)
	c.Assert(err.Struct, Equals, "TestInt32")
	c.Assert(len(err.Errors), Equals, 10)
	c.Assert(len(err.StructErrors), Equals, 0)

	// Assert Fields
	AssertFieldError(err, "Required", "required", c)
	AssertFieldError(err, "Len", "len", c)
	AssertFieldError(err, "Min", "min", c)
	AssertFieldError(err, "Max", "max", c)
	AssertFieldError(err, "MinMax", "min", c)
	AssertFieldError(err, "Lt", "lt", c)
	AssertFieldError(err, "Lte", "lte", c)
	AssertFieldError(err, "Gt", "gt", c)
	AssertFieldError(err, "Gte", "gte", c)
	AssertFieldError(err, "OmitEmpty", "max", c)
}

func (ms *MySuite) TestStructUint64Validation(c *C) {

	tSuccess := &TestUint64{
		Required:  1,
		Len:       10,
		Min:       1,
		Max:       10,
		MinMax:    5,
		OmitEmpty: 0,
	}

	err := myValidator.ValidateStruct(tSuccess)
	c.Assert(err, IsNil)

	tFail := &TestUint64{
		Required:  0,
		Len:       11,
		Min:       0,
		Max:       11,
		MinMax:    0,
		OmitEmpty: 11,
	}

	err = myValidator.ValidateStruct(tFail)

	// Assert Top Level
	c.Assert(err, NotNil)
	c.Assert(err.Struct, Equals, "TestUint64")
	c.Assert(len(err.Errors), Equals, 6)
	c.Assert(len(err.StructErrors), Equals, 0)

	// Assert Fields
	AssertFieldError(err, "Required", "required", c)
	AssertFieldError(err, "Len", "len", c)
	AssertFieldError(err, "Min", "min", c)
	AssertFieldError(err, "Max", "max", c)
	AssertFieldError(err, "MinMax", "min", c)
	AssertFieldError(err, "OmitEmpty", "max", c)
}

func (ms *MySuite) TestStructFloat64Validation(c *C) {

	tSuccess := &TestFloat64{
		Required:  1,
		Len:       10,
		Min:       1,
		Max:       10,
		MinMax:    5,
		OmitEmpty: 0,
	}

	err := myValidator.ValidateStruct(tSuccess)
	c.Assert(err, IsNil)

	tFail := &TestFloat64{
		Required:  0,
		Len:       11,
		Min:       0,
		Max:       11,
		MinMax:    0,
		OmitEmpty: 11,
	}

	err = myValidator.ValidateStruct(tFail)

	// Assert Top Level
	c.Assert(err, NotNil)
	c.Assert(err.Struct, Equals, "TestFloat64")
	c.Assert(len(err.Errors), Equals, 6)
	c.Assert(len(err.StructErrors), Equals, 0)

	// Assert Fields
	AssertFieldError(err, "Required", "required", c)
	AssertFieldError(err, "Len", "len", c)
	AssertFieldError(err, "Min", "min", c)
	AssertFieldError(err, "Max", "max", c)
	AssertFieldError(err, "MinMax", "min", c)
	AssertFieldError(err, "OmitEmpty", "max", c)
}

func (ms *MySuite) TestStructSliceValidation(c *C) {

	tSuccess := &TestSlice{
		Required:  []int{1},
		Len:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		Min:       []int{1, 2},
		Max:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		MinMax:    []int{1, 2, 3, 4, 5},
		OmitEmpty: []int{},
	}

	err := myValidator.ValidateStruct(tSuccess)
	c.Assert(err, IsNil)

	tFail := &TestSlice{
		Required:  []int{},
		Len:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
		Min:       []int{},
		Max:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
		MinMax:    []int{},
		OmitEmpty: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
	}

	err = myValidator.ValidateStruct(tFail)

	// Assert Top Level
	c.Assert(err, NotNil)
	c.Assert(err.Struct, Equals, "TestSlice")
	c.Assert(len(err.Errors), Equals, 6)
	c.Assert(len(err.StructErrors), Equals, 0)

	// Assert Fields
	AssertFieldError(err, "Required", "required", c)
	AssertFieldError(err, "Len", "len", c)
	AssertFieldError(err, "Min", "min", c)
	AssertFieldError(err, "Max", "max", c)
	AssertFieldError(err, "MinMax", "min", c)
	AssertFieldError(err, "OmitEmpty", "max", c)
}

func (ms *MySuite) TestInvalidStruct(c *C) {
	s := &SubTest{
		Test: "1",
	}

	c.Assert(func() { myValidator.ValidateStruct(s.Test) }, PanicMatches, "interface passed for validation is not a struct")
}

func (ms *MySuite) TestInvalidField(c *C) {
	s := &SubTest{
		Test: "1",
	}

	c.Assert(func() { myValidator.ValidateFieldByTag(s, "required") }, PanicMatches, "Invalid field passed to ValidateFieldWithTag")
}

func (ms *MySuite) TestInvalidTagField(c *C) {
	s := &SubTest{
		Test: "1",
	}

	c.Assert(func() { myValidator.ValidateFieldByTag(s.Test, "") }, PanicMatches, fmt.Sprintf("Invalid validation tag on field %s", ""))
}

func (ms *MySuite) TestInvalidValidatorFunction(c *C) {
	s := &SubTest{
		Test: "1",
	}

	c.Assert(func() { myValidator.ValidateFieldByTag(s.Test, "zzxxBadFunction") }, PanicMatches, fmt.Sprintf("Undefined validation function on field %s", ""))
}
