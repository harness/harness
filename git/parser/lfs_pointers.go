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
	"bytes"
	"errors"
	"regexp"
)

const lfsPointerVersionPrefix = "version https://git-lfs.github.com/spec"

var (
	regexLFSOID  = regexp.MustCompile(`(?m)^oid sha256:([a-f0-9]{64})$`)
	regexLFSSize = regexp.MustCompile(`(?m)^size [0-9]+$`)

	ErrInvalidLFSPointer = errors.New("invalid lfs pointer")
)

func GetLFSOID(content []byte) (string, error) {
	if !bytes.HasPrefix(content, []byte(lfsPointerVersionPrefix)) {
		return "", ErrInvalidLFSPointer
	}

	oidMatch := regexLFSOID.FindSubmatch(content)
	if oidMatch == nil {
		return "", ErrInvalidLFSPointer
	}

	if !regexLFSSize.Match(content) {
		return "", ErrInvalidLFSPointer
	}

	return string(oidMatch[1]), nil
}
