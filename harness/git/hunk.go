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

package git

import (
	"strconv"
	"strings"
)

type Hunk struct {
	HunkHeader
	Lines []string
}

type HunkHeader struct {
	OldLine int
	OldSpan int
	NewLine int
	NewSpan int
	Text    string
}

func (h *HunkHeader) IsZero() bool {
	return h.OldLine == 0 && h.OldSpan == 0 && h.NewLine == 0 && h.NewSpan == 0
}

func (h *HunkHeader) IsValid() bool {
	oldOk := h.OldLine == 0 && h.OldSpan == 0 || h.OldLine > 0 && h.OldSpan > 0
	newOk := h.NewLine == 0 && h.NewSpan == 0 || h.NewLine > 0 && h.NewSpan > 0
	return !h.IsZero() && oldOk && newOk
}

func (h *HunkHeader) String() string {
	sb := strings.Builder{}

	sb.WriteString("@@ -")

	sb.WriteString(strconv.Itoa(h.OldLine))
	if h.OldSpan != 1 {
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(h.OldSpan))
	}

	sb.WriteString(" +")

	sb.WriteString(strconv.Itoa(h.NewLine))
	if h.NewSpan != 1 {
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(h.NewSpan))
	}

	sb.WriteString(" @@")

	if h.Text != "" {
		sb.WriteByte(' ')
		sb.WriteString(h.Text)
	}

	return sb.String()
}
