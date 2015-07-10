package validator

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// NOTES:
// - Run "go test" to run tests
// - Run "gocov test | gocov report" to report on test converage by file
// - Run "gocov test | gocov annotate -" to report on all code and functions, those ,marked with "MISS" were never called
//
//
// go test -cpuprofile cpu.out
// ./validator.test -test.bench=. -test.cpuprofile=cpu.prof
// go tool pprof validator.test cpu.prof
//
//
// go test -memprofile mem.out

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

var validate = New("validate", BakedInValidators)

func IsEqual(t *testing.T, val1, val2 interface{}) bool {
	v1 := reflect.ValueOf(val1)
	v2 := reflect.ValueOf(val2)

	if v1.Kind() == reflect.Ptr {
		v1 = v1.Elem()
	}

	if v2.Kind() == reflect.Ptr {
		v2 = v2.Elem()
	}

	if !v1.IsValid() && !v2.IsValid() {
		return true
	}

	v1Underlying := reflect.Zero(reflect.TypeOf(v1)).Interface()
	v2Underlying := reflect.Zero(reflect.TypeOf(v2)).Interface()

	if v1 == v1Underlying {
		if v2 == v2Underlying {
			goto CASE4
		} else {
			goto CASE3
		}
	} else {
		if v2 == v2Underlying {
			goto CASE2
		} else {
			goto CASE1
		}
	}

CASE1:
	return reflect.DeepEqual(v1.Interface(), v2.Interface())

CASE2:
	return reflect.DeepEqual(v1.Interface(), v2)
CASE3:
	return reflect.DeepEqual(v1, v2.Interface())
CASE4:
	return reflect.DeepEqual(v1, v2)
}

func Equal(t *testing.T, val1, val2 interface{}) {
	EqualSkip(t, 2, val1, val2)
}

func EqualSkip(t *testing.T, skip int, val1, val2 interface{}) {

	if !IsEqual(t, val1, val2) {

		_, file, line, _ := runtime.Caller(skip)
		fmt.Printf("%s:%d %v does not equal %v\n", path.Base(file), line, val1, val2)
		t.FailNow()
	}
}

func NotEqual(t *testing.T, val1, val2 interface{}) {
	NotEqualSkip(t, 2, val1, val2)
}

func NotEqualSkip(t *testing.T, skip int, val1, val2 interface{}) {

	if IsEqual(t, val1, val2) {
		_, file, line, _ := runtime.Caller(skip)
		fmt.Printf("%s:%d %v should not be equal %v\n", path.Base(file), line, val1, val2)
		t.FailNow()
	}
}

func PanicMatches(t *testing.T, fn func(), matches string) {
	PanicMatchesSkip(t, 2, fn, matches)
}

func PanicMatchesSkip(t *testing.T, skip int, fn func(), matches string) {

	_, file, line, _ := runtime.Caller(skip)

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Sprintf("%s", r)

			if err != matches {
				fmt.Printf("%s:%d Panic...  expected [%s] received [%s]", path.Base(file), line, matches, err)
				t.FailNow()
			}
		}
	}()

	fn()
}

func AssertStruct(t *testing.T, s *StructErrors, structFieldName string, expectedStructName string) *StructErrors {

	val, ok := s.StructErrors[structFieldName]
	EqualSkip(t, 2, ok, true)
	NotEqualSkip(t, 2, val, nil)
	EqualSkip(t, 2, val.Struct, expectedStructName)

	return val
}

func AssertFieldError(t *testing.T, s *StructErrors, field string, expectedTag string) {

	val, ok := s.Errors[field]
	EqualSkip(t, 2, ok, true)
	NotEqualSkip(t, 2, val, nil)
	EqualSkip(t, 2, val.Field, field)
	EqualSkip(t, 2, val.Tag, expectedTag)
}

func AssertMapFieldError(t *testing.T, s map[string]*FieldError, field string, expectedTag string) {

	val, ok := s[field]
	EqualSkip(t, 2, ok, true)
	NotEqualSkip(t, 2, val, nil)
	EqualSkip(t, 2, val.Field, field)
	EqualSkip(t, 2, val.Tag, expectedTag)
}

func TestExcludesRuneValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"excludesrune=☻"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "a☺b☻c☹d", Tag: "excludesrune=☻", ExpectedNil: false},
		{Value: "abcd", Tag: "excludesrune=☻", ExpectedNil: true},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}
}

func TestExcludesAllValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"excludesall=@!{}[]"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "abcd@!jfk", Tag: "excludesall=@!{}[]", ExpectedNil: false},
		{Value: "abcdefg", Tag: "excludesall=@!{}[]", ExpectedNil: true},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}

	username := "joeybloggs "

	err := validate.Field(username, "excludesall=@ ")
	NotEqual(t, err, nil)

	excluded := ","

	err = validate.Field(excluded, "excludesall=!@#$%^&*()_+.0x2C?")
	NotEqual(t, err, nil)

	excluded = "="

	err = validate.Field(excluded, "excludesall=!@#$%^&*()_+.0x2C=?")
	NotEqual(t, err, nil)
}

func TestExcludesValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"excludes=@"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "abcd@!jfk", Tag: "excludes=@", ExpectedNil: false},
		{Value: "abcdq!jfk", Tag: "excludes=@", ExpectedNil: true},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}
}

func TestContainsRuneValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"containsrune=☻"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "a☺b☻c☹d", Tag: "containsrune=☻", ExpectedNil: true},
		{Value: "abcd", Tag: "containsrune=☻", ExpectedNil: false},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}
}

func TestContainsAnyValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"containsany=@!{}[]"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "abcd@!jfk", Tag: "containsany=@!{}[]", ExpectedNil: true},
		{Value: "abcdefg", Tag: "containsany=@!{}[]", ExpectedNil: false},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}
}

func TestContainsValidation(t *testing.T) {

	tests := []struct {
		Value       string `validate:"contains=@"`
		Tag         string
		ExpectedNil bool
	}{
		{Value: "abcd@!jfk", Tag: "contains=@", ExpectedNil: true},
		{Value: "abcdq!jfk", Tag: "contains=@", ExpectedNil: false},
	}

	for i, s := range tests {
		err := validate.Field(s.Value, s.Tag)

		if (s.ExpectedNil && err != nil) || (!s.ExpectedNil && err == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, err)
		}

		errs := validate.Struct(s)

		if (s.ExpectedNil && errs != nil) || (!s.ExpectedNil && errs == nil) {
			t.Fatalf("Index: %d failed Error: %s", i, errs)
		}
	}
}

func TestIsNeFieldValidation(t *testing.T) {

	var j uint64
	var k float64
	s := "abcd"
	i := 1
	j = 1
	k = 1.543
	arr := []string{"test"}
	now := time.Now().UTC()

	var j2 uint64
	var k2 float64
	s2 := "abcdef"
	i2 := 3
	j2 = 2
	k2 = 1.5434456
	arr2 := []string{"test", "test2"}
	arr3 := []string{"test"}
	now2 := now

	err := validate.FieldWithValue(s, s2, "nefield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(i2, i, "nefield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(j2, j, "nefield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(k2, k, "nefield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(arr2, arr, "nefield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(now2, now, "nefield")
	NotEqual(t, err, nil)

	err = validate.FieldWithValue(arr3, arr, "nefield")
	NotEqual(t, err, nil)

	type Test struct {
		Start *time.Time `validate:"nefield=End"`
		End   *time.Time
	}

	sv := &Test{
		Start: &now,
		End:   &now,
	}

	errs := validate.Struct(sv)
	NotEqual(t, errs, nil)

	now3 := time.Now().UTC()

	sv = &Test{
		Start: &now,
		End:   &now3,
	}

	errs = validate.Struct(sv)
	Equal(t, errs, nil)

	channel := make(chan string)

	PanicMatches(t, func() { validate.FieldWithValue(nil, 1, "nefield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(5, channel, "nefield") }, "Bad field type chan string")
	PanicMatches(t, func() { validate.FieldWithValue(5, now, "nefield") }, "Bad Top Level field type")

	type Test2 struct {
		Start *time.Time `validate:"nefield=NonExistantField"`
		End   *time.Time
	}

	sv2 := &Test2{
		Start: &now,
		End:   &now,
	}

	PanicMatches(t, func() { validate.Struct(sv2) }, "Field \"NonExistantField\" not found in struct")
}

func TestIsNeValidation(t *testing.T) {

	var j uint64
	var k float64
	s := "abcdef"
	i := 3
	j = 2
	k = 1.5434
	arr := []string{"test"}
	now := time.Now().UTC()

	err := validate.Field(s, "ne=abcd")
	Equal(t, err, nil)

	err = validate.Field(i, "ne=1")
	Equal(t, err, nil)

	err = validate.Field(j, "ne=1")
	Equal(t, err, nil)

	err = validate.Field(k, "ne=1.543")
	Equal(t, err, nil)

	err = validate.Field(arr, "ne=2")
	Equal(t, err, nil)

	err = validate.Field(arr, "ne=1")
	NotEqual(t, err, nil)

	PanicMatches(t, func() { validate.Field(now, "ne=now") }, "Bad field type time.Time")
}

func TestIsEqFieldValidation(t *testing.T) {

	var j uint64
	var k float64
	s := "abcd"
	i := 1
	j = 1
	k = 1.543
	arr := []string{"test"}
	now := time.Now().UTC()

	var j2 uint64
	var k2 float64
	s2 := "abcd"
	i2 := 1
	j2 = 1
	k2 = 1.543
	arr2 := []string{"test"}
	arr3 := []string{"test", "test2"}
	now2 := now

	err := validate.FieldWithValue(s, s2, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(i2, i, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(j2, j, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(k2, k, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(arr2, arr, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(now2, now, "eqfield")
	Equal(t, err, nil)

	err = validate.FieldWithValue(arr3, arr, "eqfield")
	NotEqual(t, err, nil)

	type Test struct {
		Start *time.Time `validate:"eqfield=End"`
		End   *time.Time
	}

	sv := &Test{
		Start: &now,
		End:   &now,
	}

	errs := validate.Struct(sv)
	Equal(t, errs, nil)

	now3 := time.Now().UTC()

	sv = &Test{
		Start: &now,
		End:   &now3,
	}

	errs = validate.Struct(sv)
	NotEqual(t, errs, nil)

	channel := make(chan string)

	PanicMatches(t, func() { validate.FieldWithValue(nil, 1, "eqfield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(5, channel, "eqfield") }, "Bad field type chan string")
	PanicMatches(t, func() { validate.FieldWithValue(5, now, "eqfield") }, "Bad Top Level field type")

	type Test2 struct {
		Start *time.Time `validate:"eqfield=NonExistantField"`
		End   *time.Time
	}

	sv2 := &Test2{
		Start: &now,
		End:   &now,
	}

	PanicMatches(t, func() { validate.Struct(sv2) }, "Field \"NonExistantField\" not found in struct")
}

func TestIsEqValidation(t *testing.T) {

	var j uint64
	var k float64
	s := "abcd"
	i := 1
	j = 1
	k = 1.543
	arr := []string{"test"}
	now := time.Now().UTC()

	err := validate.Field(s, "eq=abcd")
	Equal(t, err, nil)

	err = validate.Field(i, "eq=1")
	Equal(t, err, nil)

	err = validate.Field(j, "eq=1")
	Equal(t, err, nil)

	err = validate.Field(k, "eq=1.543")
	Equal(t, err, nil)

	err = validate.Field(arr, "eq=1")
	Equal(t, err, nil)

	err = validate.Field(arr, "eq=2")
	NotEqual(t, err, nil)

	PanicMatches(t, func() { validate.Field(now, "eq=now") }, "Bad field type time.Time")
}

func TestBase64Validation(t *testing.T) {

	s := "dW5pY29ybg=="

	err := validate.Field(s, "base64")
	Equal(t, err, nil)

	s = "dGhpIGlzIGEgdGVzdCBiYXNlNjQ="
	err = validate.Field(s, "base64")
	Equal(t, err, nil)

	s = ""
	err = validate.Field(s, "base64")
	NotEqual(t, err, nil)

	s = "dW5pY29ybg== foo bar"
	err = validate.Field(s, "base64")
	NotEqual(t, err, nil)
}

func TestStructOnlyValidation(t *testing.T) {

	type Inner struct {
		Test string `validate:"len=5"`
	}

	type Outer struct {
		InnerStruct *Inner `validate:"required,structonly"`
	}

	outer := &Outer{
		InnerStruct: nil,
	}

	errs := validate.Struct(outer).Flatten()
	NotEqual(t, errs, nil)

	inner := &Inner{
		Test: "1234",
	}

	outer = &Outer{
		InnerStruct: inner,
	}

	errs = validate.Struct(outer).Flatten()
	NotEqual(t, errs, nil)
	Equal(t, len(errs), 0)
}

func TestGtField(t *testing.T) {

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

	errs := validate.Struct(timeTest)
	Equal(t, errs, nil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := validate.Struct(timeTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "End", "gtfield")

	err3 := validate.FieldWithValue(&start, &end, "gtfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(&end, &start, "gtfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtfield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,gtfield=Val1"`
	}

	intTest := &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(intTest)
	Equal(t, errs, nil)

	intTest = &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(intTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtfield")

	err3 = validate.FieldWithValue(int(1), int(5), "gtfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(int(5), int(1), "gtfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtfield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,gtfield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(uIntTest)
	Equal(t, errs, nil)

	uIntTest = &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(uIntTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtfield")

	err3 = validate.FieldWithValue(uint(1), uint(5), "gtfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(uint(5), uint(1), "gtfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtfield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,gtfield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(floatTest)
	Equal(t, errs, nil)

	floatTest = &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(floatTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtfield")

	err3 = validate.FieldWithValue(float32(1), float32(5), "gtfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(float32(5), float32(1), "gtfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtfield")

	PanicMatches(t, func() { validate.FieldWithValue(nil, 1, "gtfield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(5, "T", "gtfield") }, "Bad field type string")
	PanicMatches(t, func() { validate.FieldWithValue(5, start, "gtfield") }, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,gtfield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &start,
		End:   &end,
	}

	PanicMatches(t, func() { validate.Struct(timeTest2) }, "Field \"NonExistantField\" not found in struct")
}

func TestLtField(t *testing.T) {

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

	errs := validate.Struct(timeTest)
	Equal(t, errs, nil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := validate.Struct(timeTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Start", "ltfield")

	err3 := validate.FieldWithValue(&end, &start, "ltfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(&start, &end, "ltfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltfield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,ltfield=Val1"`
	}

	intTest := &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(intTest)
	Equal(t, errs, nil)

	intTest = &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(intTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltfield")

	err3 = validate.FieldWithValue(int(5), int(1), "ltfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(int(1), int(5), "ltfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltfield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,ltfield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(uIntTest)
	Equal(t, errs, nil)

	uIntTest = &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(uIntTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltfield")

	err3 = validate.FieldWithValue(uint(5), uint(1), "ltfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(uint(1), uint(5), "ltfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltfield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,ltfield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(floatTest)
	Equal(t, errs, nil)

	floatTest = &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(floatTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltfield")

	err3 = validate.FieldWithValue(float32(5), float32(1), "ltfield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(float32(1), float32(5), "ltfield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltfield")

	PanicMatches(t, func() { validate.FieldWithValue(nil, 5, "ltfield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(1, "T", "ltfield") }, "Bad field type string")
	PanicMatches(t, func() { validate.FieldWithValue(1, end, "ltfield") }, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,ltfield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &end,
		End:   &start,
	}

	PanicMatches(t, func() { validate.Struct(timeTest2) }, "Field \"NonExistantField\" not found in struct")
}

func TestLteField(t *testing.T) {

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

	errs := validate.Struct(timeTest)
	Equal(t, errs, nil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := validate.Struct(timeTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Start", "ltefield")

	err3 := validate.FieldWithValue(&end, &start, "ltefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(&start, &end, "ltefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltefield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,ltefield=Val1"`
	}

	intTest := &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(intTest)
	Equal(t, errs, nil)

	intTest = &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(intTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltefield")

	err3 = validate.FieldWithValue(int(5), int(1), "ltefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(int(1), int(5), "ltefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltefield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,ltefield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(uIntTest)
	Equal(t, errs, nil)

	uIntTest = &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(uIntTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltefield")

	err3 = validate.FieldWithValue(uint(5), uint(1), "ltefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(uint(1), uint(5), "ltefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltefield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,ltefield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs = validate.Struct(floatTest)
	Equal(t, errs, nil)

	floatTest = &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs2 = validate.Struct(floatTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "ltefield")

	err3 = validate.FieldWithValue(float32(5), float32(1), "ltefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(float32(1), float32(5), "ltefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "ltefield")

	PanicMatches(t, func() { validate.FieldWithValue(nil, 5, "ltefield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(1, "T", "ltefield") }, "Bad field type string")
	PanicMatches(t, func() { validate.FieldWithValue(1, end, "ltefield") }, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,ltefield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &end,
		End:   &start,
	}

	PanicMatches(t, func() { validate.Struct(timeTest2) }, "Field \"NonExistantField\" not found in struct")
}

func TestGteField(t *testing.T) {

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

	errs := validate.Struct(timeTest)
	Equal(t, errs, nil)

	timeTest = &TimeTest{
		Start: &end,
		End:   &start,
	}

	errs2 := validate.Struct(timeTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "End", "gtefield")

	err3 := validate.FieldWithValue(&start, &end, "gtefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(&end, &start, "gtefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtefield")

	type IntTest struct {
		Val1 int `validate:"required"`
		Val2 int `validate:"required,gtefield=Val1"`
	}

	intTest := &IntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(intTest)
	Equal(t, errs, nil)

	intTest = &IntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(intTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtefield")

	err3 = validate.FieldWithValue(int(1), int(5), "gtefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(int(5), int(1), "gtefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtefield")

	type UIntTest struct {
		Val1 uint `validate:"required"`
		Val2 uint `validate:"required,gtefield=Val1"`
	}

	uIntTest := &UIntTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(uIntTest)
	Equal(t, errs, nil)

	uIntTest = &UIntTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(uIntTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtefield")

	err3 = validate.FieldWithValue(uint(1), uint(5), "gtefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(uint(5), uint(1), "gtefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtefield")

	type FloatTest struct {
		Val1 float64 `validate:"required"`
		Val2 float64 `validate:"required,gtefield=Val1"`
	}

	floatTest := &FloatTest{
		Val1: 1,
		Val2: 5,
	}

	errs = validate.Struct(floatTest)
	Equal(t, errs, nil)

	floatTest = &FloatTest{
		Val1: 5,
		Val2: 1,
	}

	errs2 = validate.Struct(floatTest).Flatten()
	NotEqual(t, errs2, nil)
	AssertMapFieldError(t, errs2, "Val2", "gtefield")

	err3 = validate.FieldWithValue(float32(1), float32(5), "gtefield")
	Equal(t, err3, nil)

	err3 = validate.FieldWithValue(float32(5), float32(1), "gtefield")
	NotEqual(t, err3, nil)
	Equal(t, err3.Tag, "gtefield")

	PanicMatches(t, func() { validate.FieldWithValue(nil, 1, "gtefield") }, "struct not passed for cross validation")
	PanicMatches(t, func() { validate.FieldWithValue(5, "T", "gtefield") }, "Bad field type string")
	PanicMatches(t, func() { validate.FieldWithValue(5, start, "gtefield") }, "Bad Top Level field type")

	type TimeTest2 struct {
		Start *time.Time `validate:"required"`
		End   *time.Time `validate:"required,gtefield=NonExistantField"`
	}

	timeTest2 := &TimeTest2{
		Start: &start,
		End:   &end,
	}

	PanicMatches(t, func() { validate.Struct(timeTest2) }, "Field \"NonExistantField\" not found in struct")
}

func TestValidateByTagAndValue(t *testing.T) {

	val := "test"
	field := "test"
	err := validate.FieldWithValue(val, field, "required")
	Equal(t, err, nil)

	fn := func(val interface{}, current interface{}, field interface{}, param string) bool {

		return current.(string) == field.(string)
	}

	validate.AddFunction("isequaltestfunc", fn)

	err = validate.FieldWithValue(val, field, "isequaltestfunc")
	Equal(t, err, nil)

	val = "unequal"

	err = validate.FieldWithValue(val, field, "isequaltestfunc")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "isequaltestfunc")
}

func TestAddFunctions(t *testing.T) {

	fn := func(val interface{}, current interface{}, field interface{}, param string) bool {

		return true
	}

	validate := New("validateme", BakedInValidators)

	err := validate.AddFunction("new", fn)
	Equal(t, err, nil)

	err = validate.AddFunction("", fn)
	NotEqual(t, err, nil)

	validate.AddFunction("new", nil)
	NotEqual(t, err, nil)

	err = validate.AddFunction("new", fn)
	Equal(t, err, nil)
}

func TestChangeTag(t *testing.T) {

	validate := New("validateme", BakedInValidators)
	validate.SetTag("val")

	type Test struct {
		Name string `val:"len=4"`
	}
	s := &Test{
		Name: "TEST",
	}

	err := validate.Struct(s)
	Equal(t, err, nil)
}

func TestUnexposedStruct(t *testing.T) {

	type Test struct {
		Name      string
		unexposed struct {
			A string `validate:"required"`
		}
	}

	s := &Test{
		Name: "TEST",
	}

	err := validate.Struct(s)
	Equal(t, err, nil)
}

func TestBadParams(t *testing.T) {

	i := 1
	err := validate.Field(i, "-")
	Equal(t, err, nil)

	PanicMatches(t, func() { validate.Field(i, "len=a") }, "strconv.ParseInt: parsing \"a\": invalid syntax")
	PanicMatches(t, func() { validate.Field(i, "len=a") }, "strconv.ParseInt: parsing \"a\": invalid syntax")

	var ui uint = 1
	PanicMatches(t, func() { validate.Field(ui, "len=a") }, "strconv.ParseUint: parsing \"a\": invalid syntax")

	f := 1.23
	PanicMatches(t, func() { validate.Field(f, "len=a") }, "strconv.ParseFloat: parsing \"a\": invalid syntax")
}

func TestLength(t *testing.T) {

	i := true
	PanicMatches(t, func() { validate.Field(i, "len") }, "Bad field type bool")
}

func TestIsGt(t *testing.T) {

	myMap := map[string]string{}
	err := validate.Field(myMap, "gt=0")
	NotEqual(t, err, nil)

	f := 1.23
	err = validate.Field(f, "gt=5")
	NotEqual(t, err, nil)

	var ui uint = 5
	err = validate.Field(ui, "gt=10")
	NotEqual(t, err, nil)

	i := true
	PanicMatches(t, func() { validate.Field(i, "gt") }, "Bad field type bool")

	tm := time.Now().UTC()
	tm = tm.Add(time.Hour * 24)

	err = validate.Field(tm, "gt")
	Equal(t, err, nil)

	t2 := time.Now().UTC()

	err = validate.Field(t2, "gt")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "gt")

	type Test struct {
		Now *time.Time `validate:"gt"`
	}
	s := &Test{
		Now: &tm,
	}

	errs := validate.Struct(s)
	Equal(t, errs, nil)

	s = &Test{
		Now: &t2,
	}

	errs = validate.Struct(s)
	NotEqual(t, errs, nil)
}

func TestIsGte(t *testing.T) {

	i := true
	PanicMatches(t, func() { validate.Field(i, "gte") }, "Bad field type bool")

	t1 := time.Now().UTC()
	t1 = t1.Add(time.Hour * 24)

	err := validate.Field(t1, "gte")
	Equal(t, err, nil)

	t2 := time.Now().UTC()

	err = validate.Field(t2, "gte")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "gte")
	Equal(t, err.Type, reflect.TypeOf(time.Time{}))

	type Test struct {
		Now *time.Time `validate:"gte"`
	}
	s := &Test{
		Now: &t1,
	}

	errs := validate.Struct(s)
	Equal(t, errs, nil)

	s = &Test{
		Now: &t2,
	}

	errs = validate.Struct(s)
	NotEqual(t, errs, nil)
}

func TestIsLt(t *testing.T) {

	myMap := map[string]string{}
	err := validate.Field(myMap, "lt=0")
	NotEqual(t, err, nil)

	f := 1.23
	err = validate.Field(f, "lt=0")
	NotEqual(t, err, nil)

	var ui uint = 5
	err = validate.Field(ui, "lt=0")
	NotEqual(t, err, nil)

	i := true
	PanicMatches(t, func() { validate.Field(i, "lt") }, "Bad field type bool")

	t1 := time.Now().UTC()

	err = validate.Field(t1, "lt")
	Equal(t, err, nil)

	t2 := time.Now().UTC()
	t2 = t2.Add(time.Hour * 24)

	err = validate.Field(t2, "lt")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "lt")

	type Test struct {
		Now *time.Time `validate:"lt"`
	}

	s := &Test{
		Now: &t1,
	}

	errs := validate.Struct(s)
	Equal(t, errs, nil)

	s = &Test{
		Now: &t2,
	}

	errs = validate.Struct(s)
	NotEqual(t, errs, nil)
}

func TestIsLte(t *testing.T) {

	i := true
	PanicMatches(t, func() { validate.Field(i, "lte") }, "Bad field type bool")

	t1 := time.Now().UTC()

	err := validate.Field(t1, "lte")
	Equal(t, err, nil)

	t2 := time.Now().UTC()
	t2 = t2.Add(time.Hour * 24)

	err = validate.Field(t2, "lte")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "lte")

	type Test struct {
		Now *time.Time `validate:"lte"`
	}

	s := &Test{
		Now: &t1,
	}

	errs := validate.Struct(s)
	Equal(t, errs, nil)

	s = &Test{
		Now: &t2,
	}

	errs = validate.Struct(s)
	NotEqual(t, errs, nil)
}

func TestUrl(t *testing.T) {

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
	for i, test := range tests {

		err := validate.Field(test.param, "url")

		if test.expected == true {
			if !IsEqual(t, err, nil) {
				t.Fatalf("Index: %d URL failed Error: %s", i, err)
			}
		} else {
			if IsEqual(t, err, nil) || !IsEqual(t, err.Tag, "url") {
				t.Fatalf("Index: %d URL failed Error: %s", i, err)
			}
		}
	}

	i := 1
	PanicMatches(t, func() { validate.Field(i, "url") }, "Bad field type int")
}

func TestUri(t *testing.T) {

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
	for i, test := range tests {

		err := validate.Field(test.param, "uri")

		if test.expected == true {
			if !IsEqual(t, err, nil) {
				t.Fatalf("Index: %d URI failed Error: %s", i, err)
			}
		} else {
			if IsEqual(t, err, nil) || !IsEqual(t, err.Tag, "uri") {
				t.Fatalf("Index: %d URI failed Error: %s", i, err)
			}
		}
	}

	i := 1
	PanicMatches(t, func() { validate.Field(i, "uri") }, "Bad field type int")
}

func TestOrTag(t *testing.T) {
	s := "rgba(0,31,255,0.5)"
	err := validate.Field(s, "rgb|rgba")
	Equal(t, err, nil)

	s = "rgba(0,31,255,0.5)"
	err = validate.Field(s, "rgb|rgba|len=18")
	Equal(t, err, nil)

	s = "this ain't right"
	err = validate.Field(s, "rgb|rgba")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgb|rgba")

	s = "this ain't right"
	err = validate.Field(s, "rgb|rgba|len=10")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgb|rgba|len")

	s = "this is right"
	err = validate.Field(s, "rgb|rgba|len=13")
	Equal(t, err, nil)

	s = ""
	err = validate.Field(s, "omitempty,rgb|rgba")
	Equal(t, err, nil)
}

func TestHsla(t *testing.T) {

	s := "hsla(360,100%,100%,1)"
	err := validate.Field(s, "hsla")
	Equal(t, err, nil)

	s = "hsla(360,100%,100%,0.5)"
	err = validate.Field(s, "hsla")
	Equal(t, err, nil)

	s = "hsla(0,0%,0%, 0)"
	err = validate.Field(s, "hsla")
	Equal(t, err, nil)

	s = "hsl(361,100%,50%,1)"
	err = validate.Field(s, "hsla")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsla")

	s = "hsl(361,100%,50%)"
	err = validate.Field(s, "hsla")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsla")

	s = "hsla(361,100%,50%)"
	err = validate.Field(s, "hsla")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsla")

	s = "hsla(360,101%,50%)"
	err = validate.Field(s, "hsla")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsla")

	s = "hsla(360,100%,101%)"
	err = validate.Field(s, "hsla")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsla")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "hsla") }, "interface conversion: interface is int, not string")
}

func TestHsl(t *testing.T) {

	s := "hsl(360,100%,50%)"
	err := validate.Field(s, "hsl")
	Equal(t, err, nil)

	s = "hsl(0,0%,0%)"
	err = validate.Field(s, "hsl")
	Equal(t, err, nil)

	s = "hsl(361,100%,50%)"
	err = validate.Field(s, "hsl")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsl")

	s = "hsl(361,101%,50%)"
	err = validate.Field(s, "hsl")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsl")

	s = "hsl(361,100%,101%)"
	err = validate.Field(s, "hsl")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsl")

	s = "hsl(-10,100%,100%)"
	err = validate.Field(s, "hsl")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hsl")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "hsl") }, "interface conversion: interface is int, not string")
}

func TestRgba(t *testing.T) {

	s := "rgba(0,31,255,0.5)"
	err := validate.Field(s, "rgba")
	Equal(t, err, nil)

	s = "rgba(0,31,255,0.12)"
	err = validate.Field(s, "rgba")
	Equal(t, err, nil)

	s = "rgba( 0,  31, 255, 0.5)"
	err = validate.Field(s, "rgba")
	Equal(t, err, nil)

	s = "rgb(0,  31, 255)"
	err = validate.Field(s, "rgba")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgba")

	s = "rgb(1,349,275,0.5)"
	err = validate.Field(s, "rgba")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgba")

	s = "rgb(01,31,255,0.5)"
	err = validate.Field(s, "rgba")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgba")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "rgba") }, "interface conversion: interface is int, not string")
}

func TestRgb(t *testing.T) {

	s := "rgb(0,31,255)"
	err := validate.Field(s, "rgb")
	Equal(t, err, nil)

	s = "rgb(0,  31, 255)"
	err = validate.Field(s, "rgb")
	Equal(t, err, nil)

	s = "rgb(1,349,275)"
	err = validate.Field(s, "rgb")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgb")

	s = "rgb(01,31,255)"
	err = validate.Field(s, "rgb")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgb")

	s = "rgba(0,31,255)"
	err = validate.Field(s, "rgb")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "rgb")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "rgb") }, "interface conversion: interface is int, not string")
}

func TestEmail(t *testing.T) {

	s := "test@mail.com"
	err := validate.Field(s, "email")
	Equal(t, err, nil)

	s = ""
	err = validate.Field(s, "email")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "email")

	s = "test@email"
	err = validate.Field(s, "email")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "email")

	s = "test@email."
	err = validate.Field(s, "email")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "email")

	s = "@email.com"
	err = validate.Field(s, "email")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "email")

	i := true
	PanicMatches(t, func() { validate.Field(i, "email") }, "interface conversion: interface is bool, not string")
}

func TestHexColor(t *testing.T) {

	s := "#fff"
	err := validate.Field(s, "hexcolor")
	Equal(t, err, nil)

	s = "#c2c2c2"
	err = validate.Field(s, "hexcolor")
	Equal(t, err, nil)

	s = "fff"
	err = validate.Field(s, "hexcolor")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hexcolor")

	s = "fffFF"
	err = validate.Field(s, "hexcolor")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hexcolor")

	i := true
	PanicMatches(t, func() { validate.Field(i, "hexcolor") }, "interface conversion: interface is bool, not string")
}

func TestHexadecimal(t *testing.T) {

	s := "ff0044"
	err := validate.Field(s, "hexadecimal")
	Equal(t, err, nil)

	s = "abcdefg"
	err = validate.Field(s, "hexadecimal")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "hexadecimal")

	i := true
	PanicMatches(t, func() { validate.Field(i, "hexadecimal") }, "interface conversion: interface is bool, not string")
}

func TestNumber(t *testing.T) {

	s := "1"
	err := validate.Field(s, "number")
	Equal(t, err, nil)

	s = "+1"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "-1"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "1.12"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "+1.12"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "-1.12"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "1."
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	s = "1.o"
	err = validate.Field(s, "number")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "number")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "number") }, "interface conversion: interface is int, not string")
}

func TestNumeric(t *testing.T) {

	s := "1"
	err := validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "+1"
	err = validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "-1"
	err = validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "1.12"
	err = validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "+1.12"
	err = validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "-1.12"
	err = validate.Field(s, "numeric")
	Equal(t, err, nil)

	s = "1."
	err = validate.Field(s, "numeric")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "numeric")

	s = "1.o"
	err = validate.Field(s, "numeric")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "numeric")

	i := 1
	PanicMatches(t, func() { validate.Field(i, "numeric") }, "interface conversion: interface is int, not string")
}

func TestAlphaNumeric(t *testing.T) {

	s := "abcd123"
	err := validate.Field(s, "alphanum")
	Equal(t, err, nil)

	s = "abc!23"
	err = validate.Field(s, "alphanum")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "alphanum")

	PanicMatches(t, func() { validate.Field(1, "alphanum") }, "interface conversion: interface is int, not string")
}

func TestAlpha(t *testing.T) {

	s := "abcd"
	err := validate.Field(s, "alpha")
	Equal(t, err, nil)

	s = "abc1"
	err = validate.Field(s, "alpha")
	NotEqual(t, err, nil)
	Equal(t, err.Tag, "alpha")

	PanicMatches(t, func() { validate.Field(1, "alpha") }, "interface conversion: interface is int, not string")
}

func TestFlattening(t *testing.T) {

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

	err1 := validate.Struct(tSuccess).Flatten()
	Equal(t, len(err1), 0)

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

	err2 := validate.Struct(tFail).Flatten()

	// Assert Top Level
	NotEqual(t, err2, nil)

	// Assert Fields
	AssertMapFieldError(t, err2, "Len", "len")
	AssertMapFieldError(t, err2, "Gt", "gt")
	AssertMapFieldError(t, err2, "Gte", "gte")

	// Assert Struct Field
	AssertMapFieldError(t, err2, "Sub.Test", "required")

	// Assert Anonymous Struct Field
	AssertMapFieldError(t, err2, "Anonymous.A", "required")

	// Assert Interface Field
	AssertMapFieldError(t, err2, "Iface.F", "len")
}

func TestStructStringValidation(t *testing.T) {

	validate.SetMaxStructPoolSize(11)

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

	err := validate.Struct(tSuccess)
	Equal(t, err, nil)

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

	err = validate.Struct(tFail)

	// Assert Top Level
	NotEqual(t, err, nil)
	Equal(t, err.Struct, "TestString")
	Equal(t, len(err.Errors), 10)
	Equal(t, len(err.StructErrors), 3)

	// Assert Fields
	AssertFieldError(t, err, "Required", "required")
	AssertFieldError(t, err, "Len", "len")
	AssertFieldError(t, err, "Min", "min")
	AssertFieldError(t, err, "Max", "max")
	AssertFieldError(t, err, "MinMax", "min")
	AssertFieldError(t, err, "Gt", "gt")
	AssertFieldError(t, err, "Gte", "gte")
	AssertFieldError(t, err, "OmitEmpty", "max")

	// Assert Anonymous embedded struct
	AssertStruct(t, err, "Anonymous", "")

	// Assert SubTest embedded struct
	val := AssertStruct(t, err, "Sub", "SubTest")
	Equal(t, len(val.Errors), 1)
	Equal(t, len(val.StructErrors), 0)

	AssertFieldError(t, val, "Test", "required")

	errors := err.Error()
	NotEqual(t, errors, nil)
}

func TestStructInt32Validation(t *testing.T) {

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

	err := validate.Struct(tSuccess)
	Equal(t, err, nil)

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

	err = validate.Struct(tFail)

	// Assert Top Level
	NotEqual(t, err, nil)
	Equal(t, err.Struct, "TestInt32")
	Equal(t, len(err.Errors), 10)
	Equal(t, len(err.StructErrors), 0)

	// Assert Fields
	AssertFieldError(t, err, "Required", "required")
	AssertFieldError(t, err, "Len", "len")
	AssertFieldError(t, err, "Min", "min")
	AssertFieldError(t, err, "Max", "max")
	AssertFieldError(t, err, "MinMax", "min")
	AssertFieldError(t, err, "Lt", "lt")
	AssertFieldError(t, err, "Lte", "lte")
	AssertFieldError(t, err, "Gt", "gt")
	AssertFieldError(t, err, "Gte", "gte")
	AssertFieldError(t, err, "OmitEmpty", "max")
}

func TestStructUint64Validation(t *testing.T) {

	tSuccess := &TestUint64{
		Required:  1,
		Len:       10,
		Min:       1,
		Max:       10,
		MinMax:    5,
		OmitEmpty: 0,
	}

	err := validate.Struct(tSuccess)
	Equal(t, err, nil)

	tFail := &TestUint64{
		Required:  0,
		Len:       11,
		Min:       0,
		Max:       11,
		MinMax:    0,
		OmitEmpty: 11,
	}

	err = validate.Struct(tFail)

	// Assert Top Level
	NotEqual(t, err, nil)
	Equal(t, err.Struct, "TestUint64")
	Equal(t, len(err.Errors), 6)
	Equal(t, len(err.StructErrors), 0)

	// Assert Fields
	AssertFieldError(t, err, "Required", "required")
	AssertFieldError(t, err, "Len", "len")
	AssertFieldError(t, err, "Min", "min")
	AssertFieldError(t, err, "Max", "max")
	AssertFieldError(t, err, "MinMax", "min")
	AssertFieldError(t, err, "OmitEmpty", "max")
}

func TestStructFloat64Validation(t *testing.T) {

	tSuccess := &TestFloat64{
		Required:  1,
		Len:       10,
		Min:       1,
		Max:       10,
		MinMax:    5,
		OmitEmpty: 0,
	}

	err := validate.Struct(tSuccess)
	Equal(t, err, nil)

	tFail := &TestFloat64{
		Required:  0,
		Len:       11,
		Min:       0,
		Max:       11,
		MinMax:    0,
		OmitEmpty: 11,
	}

	err = validate.Struct(tFail)

	// Assert Top Level
	NotEqual(t, err, nil)
	Equal(t, err.Struct, "TestFloat64")
	Equal(t, len(err.Errors), 6)
	Equal(t, len(err.StructErrors), 0)

	// Assert Fields
	AssertFieldError(t, err, "Required", "required")
	AssertFieldError(t, err, "Len", "len")
	AssertFieldError(t, err, "Min", "min")
	AssertFieldError(t, err, "Max", "max")
	AssertFieldError(t, err, "MinMax", "min")
	AssertFieldError(t, err, "OmitEmpty", "max")
}

func TestStructSliceValidation(t *testing.T) {

	tSuccess := &TestSlice{
		Required:  []int{1},
		Len:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		Min:       []int{1, 2},
		Max:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		MinMax:    []int{1, 2, 3, 4, 5},
		OmitEmpty: []int{},
	}

	err := validate.Struct(tSuccess)
	Equal(t, err, nil)

	tFail := &TestSlice{
		Required:  []int{},
		Len:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
		Min:       []int{},
		Max:       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
		MinMax:    []int{},
		OmitEmpty: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
	}

	err = validate.Struct(tFail)

	// Assert Top Level
	NotEqual(t, err, nil)
	Equal(t, err.Struct, "TestSlice")
	Equal(t, len(err.Errors), 6)
	Equal(t, len(err.StructErrors), 0)

	// Assert Fields
	AssertFieldError(t, err, "Required", "required")
	AssertFieldError(t, err, "Len", "len")
	AssertFieldError(t, err, "Min", "min")
	AssertFieldError(t, err, "Max", "max")
	AssertFieldError(t, err, "MinMax", "min")
	AssertFieldError(t, err, "OmitEmpty", "max")
}

func TestInvalidStruct(t *testing.T) {
	s := &SubTest{
		Test: "1",
	}

	PanicMatches(t, func() { validate.Struct(s.Test) }, "interface passed for validation is not a struct")
}

func TestInvalidField(t *testing.T) {
	s := &SubTest{
		Test: "1",
	}

	PanicMatches(t, func() { validate.Field(s, "required") }, "Invalid field passed to ValidateFieldWithTag")
}

func TestInvalidTagField(t *testing.T) {
	s := &SubTest{
		Test: "1",
	}

	PanicMatches(t, func() { validate.Field(s.Test, "") }, fmt.Sprintf("Invalid validation tag on field %s", ""))
}

func TestInvalidValidatorFunction(t *testing.T) {
	s := &SubTest{
		Test: "1",
	}

	PanicMatches(t, func() { validate.Field(s.Test, "zzxxBadFunction") }, fmt.Sprintf("Undefined validation function on field %s", ""))
}

func BenchmarkValidateField(b *testing.B) {
	for n := 0; n < b.N; n++ {
		validate.Field("1", "len=1")
	}
}

func BenchmarkValidateStructSimple(b *testing.B) {

	type Foo struct {
		StringValue string `validate:"min=5,max=10"`
		IntValue    int    `validate:"min=5,max=10"`
	}

	validFoo := &Foo{StringValue: "Foobar", IntValue: 7}
	invalidFoo := &Foo{StringValue: "Fo", IntValue: 3}

	for n := 0; n < b.N; n++ {
		validate.Struct(validFoo)
		validate.Struct(invalidFoo)
	}
}

// func BenchmarkTemplateParallelSimple(b *testing.B) {

// 	type Foo struct {
// 		StringValue string `validate:"min=5,max=10"`
// 		IntValue    int    `validate:"min=5,max=10"`
// 	}

// 	validFoo := &Foo{StringValue: "Foobar", IntValue: 7}
// 	invalidFoo := &Foo{StringValue: "Fo", IntValue: 3}

// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			validate.Struct(validFoo)
// 			validate.Struct(invalidFoo)
// 		}
// 	})
// }

func BenchmarkValidateStructLarge(b *testing.B) {

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

	for n := 0; n < b.N; n++ {
		validate.Struct(tSuccess)
		validate.Struct(tFail)
	}
}

// func BenchmarkTemplateParallelLarge(b *testing.B) {

// 	tFail := &TestString{
// 		Required:  "",
// 		Len:       "",
// 		Min:       "",
// 		Max:       "12345678901",
// 		MinMax:    "",
// 		Lt:        "0123456789",
// 		Lte:       "01234567890",
// 		Gt:        "1",
// 		Gte:       "1",
// 		OmitEmpty: "12345678901",
// 		Sub: &SubTest{
// 			Test: "",
// 		},
// 		Anonymous: struct {
// 			A string `validate:"required"`
// 		}{
// 			A: "",
// 		},
// 		Iface: &Impl{
// 			F: "12",
// 		},
// 	}

// 	tSuccess := &TestString{
// 		Required:  "Required",
// 		Len:       "length==10",
// 		Min:       "min=1",
// 		Max:       "1234567890",
// 		MinMax:    "12345",
// 		Lt:        "012345678",
// 		Lte:       "0123456789",
// 		Gt:        "01234567890",
// 		Gte:       "0123456789",
// 		OmitEmpty: "",
// 		Sub: &SubTest{
// 			Test: "1",
// 		},
// 		SubIgnore: &SubTest{
// 			Test: "",
// 		},
// 		Anonymous: struct {
// 			A string `validate:"required"`
// 		}{
// 			A: "1",
// 		},
// 		Iface: &Impl{
// 			F: "123",
// 		},
// 	}

// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			validate.Struct(tSuccess)
// 			validate.Struct(tFail)
// 		}
// 	})
// }
