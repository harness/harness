// Copyright 2023 Harness, Inc.
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

package render

// pagelen calculates to total number of pages given the
// page size and total count of all paginated items.
func pagelen(size, total int) int {
	quotient, remainder := total/size, total%size
	switch {
	case quotient == 0:
		return 1
	case remainder == 0:
		return quotient
	default:
		return quotient + 1
	}
}

// max returns the largest of x or y.
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
