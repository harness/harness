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

package tasklink

import (
	"regexp"
	"strconv"
)

// Reference is a reference to a task found in the text of a pull request.
type Reference struct {
	// Number is the per-repo task number (the n of TASK-n).
	Number int64
	// Closes is true if the reference follows a closing keyword (Fixes, Closes, Resolves).
	Closes bool
}

var (
	// closingRE matches a closing keyword followed by one or more task references,
	// for example "Fixes TASK-1" or "closes #TASK-2, TASK-3 and TASK-4".
	closingRE = regexp.MustCompile(
		`(?i)\b(?:fix(?:es)?|close(?:s)?|resolve(?:s)?)\b:?[ \t]+` +
			`(#?TASK-\d+(?:[ \t]*(?:,|&|\band\b)[ \t]*#?TASK-\d+)*)`)

	// mentionRE matches any task reference. The reference can't be glued to a word or dash.
	mentionRE = regexp.MustCompile(`(?:^|[^\w#-])#?TASK-(\d+)\b`)

	// numberRE extracts the task numbers of a reference list.
	numberRE = regexp.MustCompile(`TASK-(\d+)`)
)

// ParseReferences extracts the task references of the given texts (pull request title and description).
// References are returned in order of first appearance without duplicates. A task that is referenced
// both with and without a closing keyword is reported as closing.
func ParseReferences(texts ...string) []Reference {
	var (
		result []Reference
		index  = map[int64]int{}
	)

	add := func(number int64, closes bool) {
		if number <= 0 {
			return
		}
		if i, ok := index[number]; ok {
			result[i].Closes = result[i].Closes || closes
			return
		}
		index[number] = len(result)
		result = append(result, Reference{Number: number, Closes: closes})
	}

	for _, text := range texts {
		closing := map[int64]bool{}
		for _, m := range closingRE.FindAllStringSubmatch(text, -1) {
			for _, n := range numberRE.FindAllStringSubmatch(m[1], -1) {
				if number, err := strconv.ParseInt(n[1], 10, 64); err == nil {
					closing[number] = true
				}
			}
		}

		// Walk the plain mentions in order so that the result order follows the text.
		for _, m := range mentionRE.FindAllStringSubmatch(text, -1) {
			number, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				continue
			}
			add(number, closing[number])
		}
	}

	return result
}
