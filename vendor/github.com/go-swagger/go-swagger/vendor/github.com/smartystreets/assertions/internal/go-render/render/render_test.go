// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package render

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"testing"
)

func init() {
	// For testing purposes, pointers will render as "PTR" so that they are
	// deterministic.
	renderPointer = func(buf *bytes.Buffer, p uintptr) {
		buf.WriteString("PTR")
	}
}

func assertRendersLike(t *testing.T, name string, v interface{}, exp string) {
	act := Render(v)
	if act != exp {
		_, _, line, _ := runtime.Caller(1)
		t.Errorf("On line #%d, [%s] did not match expectations:\nExpected: %s\nActual  : %s\n", line, name, exp, act)
	}
}

func TestRenderList(t *testing.T) {
	t.Parallel()

	// Note that we make some of the fields exportable. This is to avoid a fun case
	// where the first reflect.Value has a read-only bit set, but follow-on values
	// do not, so recursion tests are off by one.
	type testStruct struct {
		Name string
		I    interface{}

		m string
	}

	type myStringSlice []string
	type myStringMap map[string]string
	type myIntType int
	type myStringType string

	s0 := "string0"
	s0P := &s0
	mit := myIntType(42)
	stringer := fmt.Stringer(nil)

	for i, tc := range []struct {
		a interface{}
		s string
	}{
		{nil, `nil`},
		{make(chan int), `(chan int)(PTR)`},
		{&stringer, `(*fmt.Stringer)(nil)`},
		{123, `123`},
		{"hello", `"hello"`},
		{(*testStruct)(nil), `(*render.testStruct)(nil)`},
		{(**testStruct)(nil), `(**render.testStruct)(nil)`},
		{[]***testStruct(nil), `[]***render.testStruct(nil)`},
		{testStruct{Name: "foo", I: &testStruct{Name: "baz"}},
			`render.testStruct{Name:"foo", I:(*render.testStruct){Name:"baz", I:interface{}(nil), m:""}, m:""}`},
		{[]byte(nil), `[]uint8(nil)`},
		{[]byte{}, `[]uint8{}`},
		{map[string]string(nil), `map[string]string(nil)`},
		{[]*testStruct{
			{Name: "foo"},
			{Name: "bar"},
		}, `[]*render.testStruct{(*render.testStruct){Name:"foo", I:interface{}(nil), m:""}, ` +
			`(*render.testStruct){Name:"bar", I:interface{}(nil), m:""}}`},
		{myStringSlice{"foo", "bar"}, `render.myStringSlice{"foo", "bar"}`},
		{myStringMap{"foo": "bar"}, `render.myStringMap{"foo":"bar"}`},
		{myIntType(12), `render.myIntType(12)`},
		{&mit, `(*render.myIntType)(42)`},
		{myStringType("foo"), `render.myStringType("foo")`},
		{struct {
			a int
			b string
		}{123, "foo"}, `struct { a int; b string }{123, "foo"}`},
		{[]string{"foo", "foo", "bar", "baz", "qux", "qux"},
			`[]string{"foo", "foo", "bar", "baz", "qux", "qux"}`},
		{[...]int{1, 2, 3}, `[3]int{1, 2, 3}`},
		{map[string]bool{
			"foo": true,
			"bar": false,
		}, `map[string]bool{"bar":false, "foo":true}`},
		{map[int]string{1: "foo", 2: "bar"}, `map[int]string{1:"foo", 2:"bar"}`},
		{uint32(1337), `1337`},
		{3.14, `3.14`},
		{complex(3, 0.14), `(3+0.14i)`},
		{&s0, `(*string)("string0")`},
		{&s0P, `(**string)("string0")`},
		{[]interface{}{nil, 1, 2, nil}, `[]interface{}{interface{}(nil), 1, 2, interface{}(nil)}`},
	} {
		assertRendersLike(t, fmt.Sprintf("Input #%d", i), tc.a, tc.s)
	}
}

func TestRenderRecursiveStruct(t *testing.T) {
	type testStruct struct {
		Name string
		I    interface{}
	}

	s := &testStruct{
		Name: "recursive",
	}
	s.I = s

	assertRendersLike(t, "Recursive struct", s,
		`(*render.testStruct){Name:"recursive", I:<REC(*render.testStruct)>}`)
}

func TestRenderRecursiveArray(t *testing.T) {
	a := [2]interface{}{}
	a[0] = &a
	a[1] = &a

	assertRendersLike(t, "Recursive array", &a,
		`(*[2]interface{}){<REC(*[2]interface{})>, <REC(*[2]interface{})>}`)
}

func TestRenderRecursiveMap(t *testing.T) {
	m := map[string]interface{}{}
	foo := "foo"
	m["foo"] = m
	m["bar"] = [](*string){&foo, &foo}
	v := []map[string]interface{}{m, m}

	assertRendersLike(t, "Recursive map", v,
		`[]map[string]interface{}{{`+
			`"bar":[]*string{(*string)("foo"), (*string)("foo")}, `+
			`"foo":<REC(map[string]interface{})>}, {`+
			`"bar":[]*string{(*string)("foo"), (*string)("foo")}, `+
			`"foo":<REC(map[string]interface{})>}}`)
}

func TestRenderImplicitType(t *testing.T) {
	type namedStruct struct{ a, b int }
	type namedInt int

	tcs := []struct {
		in     interface{}
		expect string
	}{
		{
			[]struct{ a, b int }{{1, 2}},
			"[]struct { a int; b int }{{1, 2}}",
		},
		{
			map[string]struct{ a, b int }{"hi": {1, 2}},
			`map[string]struct { a int; b int }{"hi":{1, 2}}`,
		},
		{
			map[namedInt]struct{}{10: {}},
			`map[render.namedInt]struct {}{10:{}}`,
		},
		{
			struct{ a, b int }{1, 2},
			`struct { a int; b int }{1, 2}`,
		},
		{
			namedStruct{1, 2},
			"render.namedStruct{a:1, b:2}",
		},
	}

	for _, tc := range tcs {
		assertRendersLike(t, reflect.TypeOf(tc.in).String(), tc.in, tc.expect)
	}
}

func ExampleInReadme() {
	type customType int
	type testStruct struct {
		S string
		V *map[string]int
		I interface{}
	}

	a := testStruct{
		S: "hello",
		V: &map[string]int{"foo": 0, "bar": 1},
		I: customType(42),
	}

	fmt.Println("Render test:")
	fmt.Printf("fmt.Printf:    %s\n", sanitizePointer(fmt.Sprintf("%#v", a)))
	fmt.Printf("render.Render: %s\n", Render(a))
	// Output: Render test:
	// fmt.Printf:    render.testStruct{S:"hello", V:(*map[string]int)(0x600dd065), I:42}
	// render.Render: render.testStruct{S:"hello", V:(*map[string]int){"bar":1, "foo":0}, I:render.customType(42)}
}

var pointerRE = regexp.MustCompile(`\(0x[a-f0-9]+\)`)

func sanitizePointer(s string) string {
	return pointerRE.ReplaceAllString(s, "(0x600dd065)")
}

type chanList []chan int

func (c chanList) Len() int      { return len(c) }
func (c chanList) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c chanList) Less(i, j int) bool {
	return reflect.ValueOf(c[i]).Pointer() < reflect.ValueOf(c[j]).Pointer()
}

func TestMapSortRendering(t *testing.T) {
	type namedMapType map[int]struct{ a int }
	type mapKey struct{ a, b int }

	chans := make(chanList, 5)
	for i := range chans {
		chans[i] = make(chan int)
	}

	tcs := []struct {
		in     interface{}
		expect string
	}{
		{
			map[uint32]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}},
			"map[uint32]struct {}{1:{}, 2:{}, 3:{}, 4:{}, 5:{}, 6:{}, 7:{}, 8:{}}",
		},
		{
			map[int8]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}},
			"map[int8]struct {}{1:{}, 2:{}, 3:{}, 4:{}, 5:{}, 6:{}, 7:{}, 8:{}}",
		},
		{
			map[uintptr]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}},
			"map[uintptr]struct {}{1:{}, 2:{}, 3:{}, 4:{}, 5:{}, 6:{}, 7:{}, 8:{}}",
		},
		{
			namedMapType{10: struct{ a int }{20}},
			"render.namedMapType{10:struct { a int }{20}}",
		},
		{
			map[mapKey]struct{}{mapKey{3, 1}: {}, mapKey{1, 3}: {}, mapKey{1, 2}: {}, mapKey{2, 1}: {}},
			"map[render.mapKey]struct {}{render.mapKey{a:1, b:2}:{}, render.mapKey{a:1, b:3}:{}, render.mapKey{a:2, b:1}:{}, render.mapKey{a:3, b:1}:{}}",
		},
		{
			map[float64]struct{}{10.5: {}, 10.15: {}, 1203: {}, 1: {}, 2: {}},
			"map[float64]struct {}{1:{}, 2:{}, 10.15:{}, 10.5:{}, 1203:{}}",
		},
		{
			map[bool]struct{}{true: {}, false: {}},
			"map[bool]struct {}{false:{}, true:{}}",
		},
		{
			map[interface{}]struct{}{1: {}, 2: {}, 3: {}, "foo": {}},
			`map[interface{}]struct {}{1:{}, 2:{}, 3:{}, "foo":{}}`,
		},
		{
			map[complex64]struct{}{1 + 2i: {}, 2 + 1i: {}, 3 + 1i: {}, 1 + 3i: {}},
			"map[complex64]struct {}{(1+2i):{}, (1+3i):{}, (2+1i):{}, (3+1i):{}}",
		},
		{
			map[chan int]string{nil: "a", chans[0]: "b", chans[1]: "c", chans[2]: "d", chans[3]: "e", chans[4]: "f"},
			`map[(chan int)]string{(chan int)(PTR):"a", (chan int)(PTR):"b", (chan int)(PTR):"c", (chan int)(PTR):"d", (chan int)(PTR):"e", (chan int)(PTR):"f"}`,
		},
	}

	for _, tc := range tcs {
		assertRendersLike(t, reflect.TypeOf(tc.in).Name(), tc.in, tc.expect)
	}
}
