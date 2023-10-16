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
	message, _ := toJSON(p)
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

	if !p.Default && len(p.Include) == 0 && len(p.Exclude) == 0 {
		return ErrPatternEmpty
	}

	return nil
}

func (p *Pattern) Matches(branchName, defaultName string) bool {
	for _, exclude := range p.Exclude {
		if patternMatches(exclude, branchName) {
			return false
		}
	}

	if p.Default && branchName == defaultName {
		return true
	}

	for _, include := range p.Include {
		if patternMatches(include, branchName) {
			return true
		}
	}

	return false
}

func patternValidate(pattern string) error {
	if pattern == "" {
		return ErrPatternEmptyPattern
	}
	_, err := doublestar.Match(pattern, "test")
	if err != nil {
		return fmt.Errorf("name pattern is invalid: %s", pattern)
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
