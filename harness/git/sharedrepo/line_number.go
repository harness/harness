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

package sharedrepo

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// lineNumberEOF indicates a line number pointing at the end of the file.
// Setting it to max Int64 ensures that any valid lineNumber is smaller than a EOF line number.
const lineNumberEOF = lineNumber(math.MaxInt64)

type lineNumber int64

func (n lineNumber) IsEOF() bool {
	return n == lineNumberEOF
}

func (n lineNumber) String() string {
	if n == lineNumberEOF {
		return "eof"
	}
	return fmt.Sprint(int64(n))
}

func parseLineNumber(raw []byte) (lineNumber, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("line number can't be empty")
	}
	if strings.EqualFold(string(raw), "eof") {
		return lineNumberEOF, nil
	}

	val, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse %q as line number", string(raw))
	}

	if val < 1 {
		return 0, fmt.Errorf("line numbering starts at 1")
	}
	if val == int64(lineNumberEOF) {
		return 0, fmt.Errorf("line numbering ends at %d", lineNumberEOF-1)
	}

	return lineNumber(val), err
}
