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

package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/harness/gitness/git/sha"
)

var reReference = regexp.MustCompile(`^([0-9a-f]+)[ |\t](.+)$`)

func ReferenceList(r io.Reader) (map[string]sha.SHA, error) {
	result := make(map[string]sha.SHA)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		parts := reReference.FindStringSubmatch(line)
		if parts == nil {
			return nil, fmt.Errorf("unexpected output of reference list: %s", line)
		}

		refSHAStr := parts[1]
		refName := parts[2]

		refSHA, err := sha.New(refSHAStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference sha (%s): %w", refSHAStr, err)
		}

		result[refName] = refSHA
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read references: %w", err)
	}

	return result, nil
}
