package flags

import (
	"testing"
	"time"
)

func expectConvert(t *testing.T, o *Option, expected string) {
	s, err := convertToString(o.value, o.tag)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	assertString(t, s, expected)
}

func TestConvertToString(t *testing.T) {
	d, _ := time.ParseDuration("1h2m4s")

	var opts = struct {
		String string `long:"string"`

		Int   int   `long:"int"`
		Int8  int8  `long:"int8"`
		Int16 int16 `long:"int16"`
		Int32 int32 `long:"int32"`
		Int64 int64 `long:"int64"`

		Uint   uint   `long:"uint"`
		Uint8  uint8  `long:"uint8"`
		Uint16 uint16 `long:"uint16"`
		Uint32 uint32 `long:"uint32"`
		Uint64 uint64 `long:"uint64"`

		Float32 float32 `long:"float32"`
		Float64 float64 `long:"float64"`

		Duration time.Duration `long:"duration"`

		Bool bool `long:"bool"`

		IntSlice    []int           `long:"int-slice"`
		IntFloatMap map[int]float64 `long:"int-float-map"`

		PtrBool   *bool       `long:"ptr-bool"`
		Interface interface{} `long:"interface"`

		Int32Base  int32  `long:"int32-base" base:"16"`
		Uint32Base uint32 `long:"uint32-base" base:"16"`
	}{
		"string",

		-2,
		-1,
		0,
		1,
		2,

		1,
		2,
		3,
		4,
		5,

		1.2,
		-3.4,

		d,
		true,

		[]int{-3, 4, -2},
		map[int]float64{-2: 4.5},

		new(bool),
		float32(5.2),

		-5823,
		4232,
	}

	p := NewNamedParser("test", Default)
	grp, _ := p.AddGroup("test group", "", &opts)

	expects := []string{
		"string",
		"-2",
		"-1",
		"0",
		"1",
		"2",

		"1",
		"2",
		"3",
		"4",
		"5",

		"1.2",
		"-3.4",

		"1h2m4s",
		"true",

		"[-3, 4, -2]",
		"{-2:4.5}",

		"false",
		"5.2",

		"-16bf",
		"1088",
	}

	for i, v := range grp.Options() {
		expectConvert(t, v, expects[i])
	}
}

func TestConvertToStringInvalidIntBase(t *testing.T) {
	var opts = struct {
		Int int `long:"int" base:"no"`
	}{
		2,
	}

	p := NewNamedParser("test", Default)
	grp, _ := p.AddGroup("test group", "", &opts)
	o := grp.Options()[0]

	_, err := convertToString(o.value, o.tag)

	if err != nil {
		err = newErrorf(ErrMarshal, "%v", err)
	}

	assertError(t, err, ErrMarshal, "strconv.ParseInt: parsing \"no\": invalid syntax")
}

func TestConvertToStringInvalidUintBase(t *testing.T) {
	var opts = struct {
		Uint uint `long:"uint" base:"no"`
	}{
		2,
	}

	p := NewNamedParser("test", Default)
	grp, _ := p.AddGroup("test group", "", &opts)
	o := grp.Options()[0]

	_, err := convertToString(o.value, o.tag)

	if err != nil {
		err = newErrorf(ErrMarshal, "%v", err)
	}

	assertError(t, err, ErrMarshal, "strconv.ParseInt: parsing \"no\": invalid syntax")
}
