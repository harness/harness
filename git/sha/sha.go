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

package sha

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/harness/gitness/errors"
)

// EmptyTree is the SHA of an empty tree.
const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

var (
	Nil = Must("0000000000000000000000000000000000000000")
	// regex defines the valid SHA format accepted by GIT (full form and short forms).
	// Note: as of now SHA is at most 40 characters long, but in the future it's moving to sha256
	// which is 64 chars - keep this forward-compatible.
	regex    = regexp.MustCompile("^[0-9a-f]{4,64}$")
	nilRegex = regexp.MustCompile("^0{4,64}$")
)

// SHA a git commit name.
type SHA struct {
	str string
}

func New(value string) (SHA, error) {
	s := strings.TrimSpace(value)
	if !isValidGitSHA(s) {
		return SHA{}, errors.InvalidArgument("the provided commit sha '%s' is of invalid format.", s)
	}
	return SHA{
		str: s,
	}, nil
}

func (s *SHA) UnmarshalJSON(content []byte) error {
	if s == nil {
		return nil
	}
	var str string
	err := json.Unmarshal(content, &str)
	if err != nil {
		return err
	}
	s.str = str
	return nil
}

func (s *SHA) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return []byte("\"" + s.str + "\""), nil
}

// String returns string representation of the SHA.
func (s *SHA) String() string {
	if s == nil {
		return ""
	}
	return s.str
}

// IsNil returns whether this SHA1 is all zeroes.
func (s *SHA) IsNil() bool {
	// regex check (minimal length 7)
	return nilRegex.MatchString(s.str)
}

// IsEmpty returns whether this SHA1 is all zeroes.
func (s *SHA) IsEmpty() bool {
	return s == nil || s.str == ""
}

// Equal returns true if val has the same SHA.
func (s *SHA) Equal(val SHA) bool {
	if s == nil {
		return false
	}
	return s.str == val.str
}

type Constraint interface {
	~string | ~[]byte
}

func Must(value string) SHA {
	sha, err := New(value)
	if err != nil {
		panic("invalid SHA" + err.Error())
	}
	return sha
}

// isValidGitSHA returns true iff the provided string is a valid git sha (short or long form).
func isValidGitSHA(sha string) bool {
	return regex.MatchString(sha)
}
