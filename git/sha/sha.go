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
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/harness/gitness/errors"

	"github.com/swaggest/jsonschema-go"
)

var (
	Nil  = Must("0000000000000000000000000000000000000000")
	None = SHA{}
	// regex defines the valid SHA format accepted by GIT (full form and short forms).
	// Note: as of now SHA is at most 40 characters long, but in the future it's moving to sha256
	// which is 64 chars - keep this forward-compatible.
	regex    = regexp.MustCompile("^[0-9a-f]{4,64}$")
	nilRegex = regexp.MustCompile("^0{4,64}$")

	// EmptyTree is the SHA of an empty tree.
	EmptyTree = Must("4b825dc642cb6eb9a060e54bf8d69288fbee4904")
)

// SHA represents a git sha.
type SHA struct {
	str string
}

// New creates and validates SHA from the value.
func New(value string) (SHA, error) {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	if !regex.MatchString(value) {
		return SHA{}, errors.InvalidArgumentf("the provided commit sha '%s' is of invalid format.", value)
	}
	return SHA{
		str: value,
	}, nil
}

// NewOrEmpty returns None if value is empty otherwise it will try to create
// and validate new SHA object.
func NewOrEmpty(value string) (SHA, error) {
	if value == "" {
		return None, nil
	}
	return New(value)
}

func (s SHA) GobEncode() ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(s.str)
	if err != nil {
		return nil, fmt.Errorf("failed to pack sha value: %w", err)
	}
	return buffer.Bytes(), nil
}

func (s *SHA) GobDecode(v []byte) error {
	if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&s.str); err != nil {
		return fmt.Errorf("failed to unpack sha value: %w", err)
	}
	return nil
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

	sha, err := NewOrEmpty(str)
	if err != nil {
		return err
	}
	s.str = sha.str
	return nil
}

func (s SHA) MarshalJSON() ([]byte, error) {
	return []byte("\"" + s.str + "\""), nil
}

// String returns string representation of the SHA.
func (s SHA) String() string {
	return s.str
}

// IsNil returns whether this SHA is all zeroes.
func (s SHA) IsNil() bool {
	return nilRegex.MatchString(s.str)
}

// IsEmpty returns whether this SHA is empty string.
func (s SHA) IsEmpty() bool {
	return s.str == ""
}

// Equal returns true if val has the same SHA.
func (s SHA) Equal(val SHA) bool {
	return s.str == val.str
}

// Must returns sha if there is an error it will panic.
func Must(value string) SHA {
	sha, err := New(value)
	if err != nil {
		panic("invalid SHA" + err.Error())
	}
	return sha
}

func (s SHA) JSONSchema() (jsonschema.Schema, error) {
	var schema jsonschema.Schema

	schema.AddType(jsonschema.String)
	schema.WithDescription("Git object hash")

	return schema, nil
}

func (s SHA) Value() (driver.Value, error) {
	return s.str, nil
}
