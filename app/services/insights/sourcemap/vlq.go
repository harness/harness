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

// Package sourcemap parses Source Map v3 files (including index maps) and uses
// them to symbolicate minified JavaScript stack frames.
//
// Conventions (the same everywhere in this package):
//   - Lines are 1-based, like Sentry frames and editors.
//   - Columns are 0-based inside Map (as in the Source Map spec). Only
//     Symbolicate converts from/to the 1-based colno used by Sentry frames.
package sourcemap

import "errors"

// ErrInvalidVLQ is returned when a mappings string contains invalid Base64 VLQ data.
var ErrInvalidVLQ = errors.New("sourcemap: invalid base64 vlq")

const (
	vlqContinuation = 0x20
	vlqBaseMask     = 0x1f
	vlqBaseShift    = 5
	// vlqMaxShift bounds a value to 32 bits (7 groups of 5 bits).
	vlqMaxShift = 35
)

var b64Index = func() [256]int8 {
	var t [256]int8
	for i := range t {
		t[i] = -1
	}
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	for i := 0; i < len(chars); i++ {
		t[chars[i]] = int8(i)
	}
	return t
}()

// decodeVLQ decodes one signed Base64 VLQ value from s starting at pos and
// returns the value and the position after it.
func decodeVLQ(s string, pos int) (int, int, error) {
	var result uint64
	shift := uint(0)
	for {
		if pos >= len(s) {
			return 0, pos, ErrInvalidVLQ
		}
		d := b64Index[s[pos]]
		pos++
		if d < 0 || shift >= vlqMaxShift {
			return 0, pos, ErrInvalidVLQ
		}
		result += uint64(d&vlqBaseMask) << shift
		shift += vlqBaseShift
		if d&vlqContinuation == 0 {
			break
		}
	}
	if result > 1<<32 {
		return 0, pos, ErrInvalidVLQ
	}
	v := int(result >> 1)
	if result&1 == 1 {
		v = -v
	}
	return v, pos, nil
}
