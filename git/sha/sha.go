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
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/harness/gitness/errors"
)

// Nil defines empty git SHA.
var Nil = ForceNew("0000000000000000000000000000000000000000")

// EmptyTree is the SHA of an empty tree.
const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

var (
	ErrTypeNotSupported = errors.New("type not supported")
	// gitSHARegex defines the valid SHA format accepted by GIT (full form and short forms).
	// Note: as of now SHA is at most 40 characters long, but in the future it's moving to sha256
	// which is 64 chars - keep this forward-compatible.
	gitSHARegex = regexp.MustCompile("^[0-9a-f]{4,64}$")
)

// SHA a git commit name.
type SHA struct {
	bytes []byte
}

// New creates a new SHA from a value T.
func New[T Constraint](value T) (SHA, error) {
	switch arg := any(value).(type) {
	case string:
		s := strings.TrimSpace(arg)
		if !isValidGitSHA(s) {
			return SHA{}, errors.InvalidArgument("the provided commit sha '%s' is of invalid format.", s)
		}
		return SHA{
			bytes: []byte(s),
		}, nil
	case []byte:
		arg = bytes.TrimSpace(arg)
		id := make([]byte, len(arg))
		copy(id, arg)
		return SHA{bytes: id}, nil
	default:
		return SHA{}, ErrTypeNotSupported
	}
}

func (s *SHA) UnmarshalJSON(content []byte) error {
	var str string
	err := json.Unmarshal(content, &str)
	if err != nil {
		return err
	}
	n, err := New(str)
	if err != nil {
		return err
	}
	s.bytes = n.bytes
	return nil
}

func (s SHA) MarshalJSON() ([]byte, error) {
	if s.bytes == nil {
		return []byte("null"), nil
	}
	return []byte("\"" + s.String() + "\""), nil
}

// String returns string (hex) representation of the SHA.
func (s SHA) String() string {
	return string(s.bytes)
}

// IsZero returns whether this SHA1 is all zeroes.
func (s SHA) IsZero() bool {
	return len(s.bytes) == 0
}

// Equal returns true if val has the same SHA as s. It supports
// string, []byte, and SHA.
func (s SHA) Equal(val any) bool {
	switch v := val.(type) {
	case string:
		v = strings.TrimSpace(v)
		return v == s.String()
	case []byte:
		v = bytes.TrimSpace(v)
		return bytes.Equal(v, s.bytes)
	case SHA:
		return bytes.Equal(v.bytes, s.bytes)
	default:
		return false
	}
}

type Constraint interface {
	~string | ~[]byte
}

func ForceNew[T Constraint](value T) SHA {
	sha, _ := New(value)
	return sha
}

// isValidGitSHA returns true iff the provided string is a valid git sha (short or long form).
func isValidGitSHA(sha string) bool {
	return gitSHARegex.MatchString(sha)
}
