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

package logutil

import (
	"bufio"
	"fmt"
	"strings"
)

type reusableScanner struct {
	scanner *bufio.Scanner
	reader  *strings.Reader
}

func newReusableScanner() *reusableScanner {
	reader := strings.NewReader("")
	scanner := bufio.NewScanner(reader)
	return &reusableScanner{
		scanner: scanner,
		reader:  reader,
	}
}

func (r *reusableScanner) scan(input string) ([]string, error) {
	r.reader.Reset(input)
	var lines []string

	for r.scanner.Scan() {
		lines = append(lines, r.scanner.Text())
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading string %s: %w", input, err)
	}

	return lines, nil
}
