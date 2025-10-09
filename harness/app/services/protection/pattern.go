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

package protection

import (
	"encoding/json"
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

type Pattern struct {
	Default bool     `json:"default,omitempty"`
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

func (p *Pattern) JSON() json.RawMessage {
	message, _ := ToJSON(p)
	return message
}

func (p *Pattern) Validate() error {
	for _, pattern := range p.Include {
		if err := patternValidate(pattern); err != nil {
			return err
		}
	}

	for _, pattern := range p.Exclude {
		if err := patternValidate(pattern); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pattern) Matches(branchName, defaultName string) bool {
	// Initially match everything, unless the default is set or the include patterns are defined.
	matches := !p.Default && len(p.Include) == 0

	// Apply the default branch.
	matches = matches || p.Default && branchName == defaultName

	// Apply the include patterns.
	if !matches {
		for _, include := range p.Include {
			if matches = patternMatches(include, branchName); matches {
				break
			}
		}
	}

	// Apply the exclude patterns.
	for _, exclude := range p.Exclude {
		matches = matches && !patternMatches(exclude, branchName)
	}

	return matches
}

func patternValidate(pattern string) error {
	if pattern == "" {
		return ErrPatternEmpty
	}
	_, err := doublestar.Match(pattern, "test")
	if err != nil {
		return ErrInvalidGlobstarPattern
	}
	return nil
}

// patternMatches matches a name against the provided file name pattern. From the doublestar library:
//
// The pattern syntax is:
//
// pattern:
//
//	{ term }
//
// term:
//
//	'*'         matches any sequence of non-path-separators
//	'**'        matches any sequence of characters, including
//	            path separators.
//	'?'         matches any single non-path-separator character
//	'[' [ '^' ] { character-range } ']'
//	      character class (must be non-empty)
//	'{' { term } [ ',' { term } ... ] '}'
//	c           matches character c (c != '*', '?', '\\', '[')
//	'\\' c      matches character c
//
// character-range:
//
//	c           matches character c (c != '\\', '-', ']')
//	'\\' c      matches character c
//	lo '-' hi   matches character c for lo <= c <= hi
func patternMatches(pattern, branchName string) bool {
	ok, _ := doublestar.Match(pattern, branchName)
	return ok
}

func matchesRef(rawPattern json.RawMessage, defaultRef, ref string) (bool, error) {
	pattern := Pattern{}

	if err := json.Unmarshal(rawPattern, &pattern); err != nil {
		return false, fmt.Errorf("failed to parse ref pattern: %w", err)
	}

	return pattern.Matches(ref, defaultRef), nil
}

func matchesRefs(rawPattern json.RawMessage, defaultRef string, refs ...string) ([]string, error) {
	pattern := Pattern{}

	if err := json.Unmarshal(rawPattern, &pattern); err != nil {
		return nil, fmt.Errorf("failed to parse ref pattern: %w", err)
	}

	matched := make([]string, 0, len(refs))

	for _, ref := range refs {
		if pattern.Matches(ref, defaultRef) {
			matched = append(matched, ref)
		}
	}

	return matched, nil
}
