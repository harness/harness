//  Copyright 2023 Harness, Inc.
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

package utils

import (
	"reflect"
	"strings"
)

func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func HasAnySuffix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasSuffix(s, prefix) {
			return true
		}
	}
	return false
}

func SafeUint64(n int) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

func IsEmpty(slice interface{}) bool {
	if slice == nil {
		return true
	}
	val := reflect.ValueOf(slice)

	// Check if the input is a pointer
	if val.Kind() == reflect.Ptr {
		// Dereference the pointer
		val = val.Elem()
	}

	// Check if the dereferenced value is nil
	if !val.IsValid() {
		return true
	}

	return val.Len() == 0
}
