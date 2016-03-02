package sqlite3

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestCallbackArgCast(t *testing.T) {
	intConv := callbackSyntheticForTests(reflect.ValueOf(int64(math.MaxInt64)), nil)
	floatConv := callbackSyntheticForTests(reflect.ValueOf(float64(math.MaxFloat64)), nil)
	errConv := callbackSyntheticForTests(reflect.Value{}, errors.New("test"))

	tests := []struct {
		f callbackArgConverter
		o reflect.Value
	}{
		{intConv, reflect.ValueOf(int8(-1))},
		{intConv, reflect.ValueOf(int16(-1))},
		{intConv, reflect.ValueOf(int32(-1))},
		{intConv, reflect.ValueOf(uint8(math.MaxUint8))},
		{intConv, reflect.ValueOf(uint16(math.MaxUint16))},
		{intConv, reflect.ValueOf(uint32(math.MaxUint32))},
		// Special case, int64->uint64 is only 1<<63 - 1, not 1<<64 - 1
		{intConv, reflect.ValueOf(uint64(math.MaxInt64))},
		{floatConv, reflect.ValueOf(float32(math.Inf(1)))},
	}

	for _, test := range tests {
		conv := callbackArgCast{test.f, test.o.Type()}
		val, err := conv.Run(nil)
		if err != nil {
			t.Errorf("Couldn't convert to %s: %s", test.o.Type(), err)
		} else if !reflect.DeepEqual(val.Interface(), test.o.Interface()) {
			t.Errorf("Unexpected result from converting to %s: got %v, want %v", test.o.Type(), val.Interface(), test.o.Interface())
		}
	}

	conv := callbackArgCast{errConv, reflect.TypeOf(int8(0))}
	_, err := conv.Run(nil)
	if err == nil {
		t.Errorf("Expected error during callbackArgCast, but got none")
	}
}

func TestCallbackConverters(t *testing.T) {
	tests := []struct {
		v   interface{}
		err bool
	}{
		// Unfortunately, we can't tell which converter was returned,
		// but we can at least check which types can be converted.
		{[]byte{0}, false},
		{"text", false},
		{true, false},
		{int8(0), false},
		{int16(0), false},
		{int32(0), false},
		{int64(0), false},
		{uint8(0), false},
		{uint16(0), false},
		{uint32(0), false},
		{uint64(0), false},
		{int(0), false},
		{uint(0), false},
		{float64(0), false},
		{float32(0), false},

		{func() {}, true},
		{complex64(complex(0, 0)), true},
		{complex128(complex(0, 0)), true},
		{struct{}{}, true},
		{map[string]string{}, true},
		{[]string{}, true},
		{(*int8)(nil), true},
		{make(chan int), true},
	}

	for _, test := range tests {
		_, err := callbackArg(reflect.TypeOf(test.v))
		if test.err && err == nil {
			t.Errorf("Expected an error when converting %s, got no error", reflect.TypeOf(test.v))
		} else if !test.err && err != nil {
			t.Errorf("Expected converter when converting %s, got error: %s", reflect.TypeOf(test.v), err)
		}
	}

	for _, test := range tests {
		_, err := callbackRet(reflect.TypeOf(test.v))
		if test.err && err == nil {
			t.Errorf("Expected an error when converting %s, got no error", reflect.TypeOf(test.v))
		} else if !test.err && err != nil {
			t.Errorf("Expected converter when converting %s, got error: %s", reflect.TypeOf(test.v), err)
		}
	}
}
