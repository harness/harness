// Copyright 2015 Aaron Jacobs. All Rights Reserved.
// Author: aaronjjacobs@gmail.com (Aaron Jacobs)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generate

import (
	"io"
	"reflect"
	"testing"
	"unsafe"

	"github.com/smartystreets/assertions/internal/oglemock/createmock/testdata/gcs"
	. "github.com/smartystreets/assertions/internal/ogletest"
)

func TestTypeString(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TypeStringTest struct {
}

func init() { RegisterTestSuite(&TypeStringTest{}) }

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *TypeStringTest) TestCases() {
	const gcsPkgPath = "github.com/smartystreets/assertions/internal/oglemock/createmock/testdata/gcs"
	to := reflect.TypeOf

	testCases := []struct {
		t        reflect.Type
		pkgPath  string
		expected string
	}{
		/////////////////////////
		// Scalar types
		/////////////////////////

		0: {to(true), "", "bool"},
		1: {to(true), "some/pkg", "bool"},
		2: {to(int(17)), "some/pkg", "int"},
		3: {to(int32(17)), "some/pkg", "int32"},
		4: {to(uint(17)), "some/pkg", "uint"},
		5: {to(uint32(17)), "some/pkg", "uint32"},
		6: {to(uintptr(17)), "some/pkg", "uintptr"},
		7: {to(float32(17)), "some/pkg", "float32"},
		8: {to(complex64(17)), "some/pkg", "complex64"},

		/////////////////////////
		// Structs
		/////////////////////////

		9:  {to(gcs.Object{}), "some/pkg", "gcs.Object"},
		10: {to(gcs.Object{}), gcsPkgPath, "Object"},

		11: {
			to(struct {
				a int
				b gcs.Object
			}{}),
			"some/pkg",
			"struct { a int; b gcs.Object }",
		},

		12: {
			to(struct {
				a int
				b gcs.Object
			}{}),
			gcsPkgPath,
			"struct { a int; b Object }",
		},

		/////////////////////////
		// Pointers
		/////////////////////////

		13: {to((*int)(nil)), gcsPkgPath, "*int"},
		14: {to((*gcs.Object)(nil)), "some/pkg", "*gcs.Object"},
		15: {to((*gcs.Object)(nil)), gcsPkgPath, "*Object"},

		/////////////////////////
		// Arrays
		/////////////////////////

		16: {to([3]int{}), "some/pkg", "[3]int"},
		17: {to([3]gcs.Object{}), gcsPkgPath, "[3]Object"},

		/////////////////////////
		// Channels
		/////////////////////////

		18: {to((chan int)(nil)), "some/pkg", "chan int"},
		19: {to((<-chan int)(nil)), "some/pkg", "<-chan int"},
		20: {to((chan<- int)(nil)), "some/pkg", "chan<- int"},
		21: {to((<-chan gcs.Object)(nil)), gcsPkgPath, "<-chan Object"},

		/////////////////////////
		// Functions
		/////////////////////////

		22: {
			to(func(int, gcs.Object) {}),
			gcsPkgPath,
			"func(int, Object) ()",
		},

		23: {
			to(func() (*gcs.Object, error) { return nil, nil }),
			gcsPkgPath,
			"func() (*Object, error)",
		},

		24: {
			to(func(int, gcs.Object) (*gcs.Object, error) { return nil, nil }),
			gcsPkgPath,
			"func(int, Object) (*Object, error)",
		},

		/////////////////////////
		// Interfaces
		/////////////////////////

		25: {to((*error)(nil)).Elem(), "some/pkg", "error"},
		26: {to((*io.Reader)(nil)).Elem(), "some/pkg", "io.Reader"},
		27: {to((*io.Reader)(nil)).Elem(), "io", "Reader"},

		28: {
			to((*interface{})(nil)).Elem(),
			"some/pkg",
			"interface {  }",
		},

		29: {
			to((*interface {
				Foo(int)
				Bar(gcs.Object)
			})(nil)).Elem(),
			"some/pkg",
			"interface { Bar(gcs.Object) (); Foo(int) () }",
		},

		30: {
			to((*interface {
				Foo(int)
				Bar(gcs.Object)
			})(nil)).Elem(),
			gcsPkgPath,
			"interface { Bar(Object) (); Foo(int) () }",
		},

		/////////////////////////
		// Maps
		/////////////////////////

		31: {to(map[*gcs.Object]gcs.Object{}), gcsPkgPath, "map[*Object]Object"},

		/////////////////////////
		// Slices
		/////////////////////////

		32: {to([]int{}), "some/pkg", "[]int"},
		33: {to([]gcs.Object{}), gcsPkgPath, "[]Object"},

		/////////////////////////
		// Strings
		/////////////////////////

		34: {to(""), gcsPkgPath, "string"},

		/////////////////////////
		// Unsafe pointer
		/////////////////////////

		35: {to(unsafe.Pointer(nil)), gcsPkgPath, "unsafe.Pointer"},

		/////////////////////////
		// Other named types
		/////////////////////////

		36: {to(gcs.Int(17)), "some/pkg", "gcs.Int"},
		37: {to(gcs.Int(17)), gcsPkgPath, "Int"},

		38: {to(gcs.Array{}), "some/pkg", "gcs.Array"},
		39: {to(gcs.Array{}), gcsPkgPath, "Array"},

		40: {to(gcs.Chan(nil)), "some/pkg", "gcs.Chan"},
		41: {to(gcs.Chan(nil)), gcsPkgPath, "Chan"},

		42: {to(gcs.Ptr(nil)), "some/pkg", "gcs.Ptr"},
		43: {to(gcs.Ptr(nil)), gcsPkgPath, "Ptr"},

		44: {to((*gcs.Int)(nil)), "some/pkg", "*gcs.Int"},
		45: {to((*gcs.Int)(nil)), gcsPkgPath, "*Int"},
	}

	for i, tc := range testCases {
		ExpectEq(
			tc.expected,
			typeString(tc.t, tc.pkgPath),
			"Case %d: %v, %q", i, tc.t, tc.pkgPath)
	}
}
