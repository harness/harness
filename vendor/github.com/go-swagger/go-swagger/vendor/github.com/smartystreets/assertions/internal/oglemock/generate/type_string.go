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
	"fmt"
	"log"
	"reflect"
	"strings"
)

// Return the string that should be used to refer to the supplied type within
// the given package. The output is not guaranteed to be pretty, and should be
// run through a tool like gofmt afterward.
//
// For example, a pointer to an io.Reader may be rendered as "*Reader" or
// "*io.Reader" depending on whether the package path is "io" or not.
func typeString(
	t reflect.Type,
	pkgPath string) (s string) {
	// Is this type named? If so we use its name, possibly with a package prefix.
	//
	// Examples:
	//
	//     int
	//     string
	//     error
	//     gcs.Bucket
	//
	if t.Name() != "" {
		if t.PkgPath() == pkgPath {
			s = t.Name()
		} else {
			s = t.String()
		}

		return
	}

	// This type is unnamed. Recurse.
	switch t.Kind() {
	case reflect.Array:
		s = fmt.Sprintf("[%d]%s", t.Len(), typeString(t.Elem(), pkgPath))

	case reflect.Chan:
		s = fmt.Sprintf("%s %s", t.ChanDir(), typeString(t.Elem(), pkgPath))

	case reflect.Func:
		s = typeString_Func(t, pkgPath)

	case reflect.Interface:
		s = typeString_Interface(t, pkgPath)

	case reflect.Map:
		s = fmt.Sprintf(
			"map[%s]%s",
			typeString(t.Key(), pkgPath),
			typeString(t.Elem(), pkgPath))

	case reflect.Ptr:
		s = fmt.Sprintf("*%s", typeString(t.Elem(), pkgPath))

	case reflect.Slice:
		s = fmt.Sprintf("[]%s", typeString(t.Elem(), pkgPath))

	case reflect.Struct:
		s = typeString_Struct(t, pkgPath)

	default:
		log.Panicf("Unhandled kind %v for type: %v", t.Kind(), t)
	}

	return
}

func typeString_FuncOrMethod(
	name string,
	t reflect.Type,
	pkgPath string) (s string) {
	// Deal with input types.
	var in []string
	for i := 0; i < t.NumIn(); i++ {
		in = append(in, typeString(t.In(i), pkgPath))
	}

	// And output types.
	var out []string
	for i := 0; i < t.NumOut(); i++ {
		out = append(out, typeString(t.Out(i), pkgPath))
	}

	// Put it all together.
	s = fmt.Sprintf(
		"%s(%s) (%s)",
		name,
		strings.Join(in, ", "),
		strings.Join(out, ", "))

	return
}

func typeString_Func(
	t reflect.Type,
	pkgPath string) (s string) {
	return typeString_FuncOrMethod("func", t, pkgPath)
}

func typeString_Struct(
	t reflect.Type,
	pkgPath string) (s string) {
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fString := fmt.Sprintf("%s %s", f.Name, typeString(f.Type, pkgPath))
		fields = append(fields, fString)
	}

	s = fmt.Sprintf("struct { %s }", strings.Join(fields, "; "))
	return
}

func typeString_Interface(
	t reflect.Type,
	pkgPath string) (s string) {
	var methods []string
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		mString := typeString_FuncOrMethod(m.Name, m.Type, pkgPath)
		methods = append(methods, mString)
	}

	s = fmt.Sprintf("interface { %s }", strings.Join(methods, "; "))
	return
}
