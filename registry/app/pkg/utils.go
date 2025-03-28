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

package pkg

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	numberRegex = regexp.MustCompile(`\d+`)
)

func IsEmpty(slice interface{}) bool {
	if slice == nil {
		return true
	}
	return reflect.ValueOf(slice).Len() == 0
}

func JoinWithSeparator(sep string, args ...string) string {
	return strings.Join(args, sep)
}

func ExtractFirstNumber(input string) (int, error) {
	match := numberRegex.FindString(input)

	if match == "" {
		return 0, fmt.Errorf("no number found in input: %s", input)
	}

	result, err := strconv.Atoi(match)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to number: %w", match, err)
	}

	return result, nil
}
