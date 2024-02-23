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

package api

import (
	"bytes"
	"strconv"
	"time"
)

const (
	// GitTimeLayout is the (default) time layout used by git.
	GitTimeLayout = "Mon Jan _2 15:04:05 2006 -0700"
)

func NewSignatureFromCommitLine(line []byte) *Signature {
	emailStart := bytes.LastIndexByte(line, '<')
	emailEnd := bytes.LastIndexByte(line, '>')
	if emailStart == -1 || emailEnd == -1 || emailEnd < emailStart {
		return nil
	}

	sig := &Signature{
		Identity: Identity{
			Name:  string(line[:emailStart-1]),
			Email: string(line[emailStart+1 : emailEnd]),
		},
	}

	hasTime := emailEnd+2 < len(line)
	if !hasTime {
		return sig
	}

	// Check date format.
	firstChar := line[emailEnd+2]
	if firstChar >= 48 && firstChar <= 57 {
		idx := bytes.IndexByte(line[emailEnd+2:], ' ')
		if idx < 0 {
			return sig
		}

		timestring := string(line[emailEnd+2 : emailEnd+2+idx])
		seconds, _ := strconv.ParseInt(timestring, 10, 64)
		sig.When = time.Unix(seconds, 0)

		idx += emailEnd + 3
		if idx >= len(line) || idx+5 > len(line) {
			return sig
		}

		timezone := string(line[idx : idx+5])
		tzhours, err1 := strconv.ParseInt(timezone[0:3], 10, 64)
		tzmins, err2 := strconv.ParseInt(timezone[3:], 10, 64)
		if err1 != nil || err2 != nil {
			return sig
		}
		if tzhours < 0 {
			tzmins *= -1
		}
		tz := time.FixedZone("", int(tzhours*60*60+tzmins*60))
		sig.When = sig.When.In(tz)
	} else {
		t, err := time.Parse(GitTimeLayout, string(line[emailEnd+2:]))
		if err != nil {
			return sig
		}
		sig.When = t
	}
	return sig
}
