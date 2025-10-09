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

package enum

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func Sanitize[E constraints.Ordered](element E, all func() ([]E, E)) (E, bool) {
	allValues, defValue := all()
	var empty E
	if element == empty && defValue != empty {
		return defValue, true
	}
	idx, exists := slices.BinarySearch(allValues, element)
	if exists {
		return allValues[idx], true
	}
	return defValue, false
}
