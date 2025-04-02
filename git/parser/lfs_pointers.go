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
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"
)

// LfsPointerMaxSize is the maximum size for an LFS pointer file.
// This is used to identify blobs that are too large to be valid LFS pointers.
// lfs-pointer specification ref: https://github.com/git-lfs/git-lfs/blob/master/docs/spec.md#the-pointer
const LfsPointerMaxSize = 200

const lfsPointerVersionPrefix = "version https://git-lfs.github.com/spec"

type LFSPointer struct {
	OID  string
	Size int64
}

var (
	regexLFSOID  = regexp.MustCompile(`(?m)^oid sha256:([a-f0-9]{64})$`)
	regexLFSSize = regexp.MustCompile(`(?m)^size (\d+)+$`)

	ErrInvalidLFSPointer = errors.New("invalid lfs pointer")
)

func GetLFSObjectID(content []byte) (string, error) {
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

func IsLFSPointer(
	ctx context.Context,
	content []byte,
	size int64,
) (*LFSPointer, bool) {
	if size > LfsPointerMaxSize {
		return nil, false
	}

	if !bytes.HasPrefix(content, []byte(lfsPointerVersionPrefix)) {
		return nil, false
	}

	oidMatch := regexLFSOID.FindSubmatch(content)
	if oidMatch == nil {
		return nil, false
	}

	sizeMatch := regexLFSSize.FindSubmatch(content)
	if sizeMatch == nil {
		return nil, false
	}

	contentSize, err := strconv.ParseInt(string(sizeMatch[1]), 10, 64)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to parse lfs pointer size for object ID %s", oidMatch[1])
		return nil, false
	}

	return &LFSPointer{OID: string(oidMatch[1]), Size: contentSize}, true
}
