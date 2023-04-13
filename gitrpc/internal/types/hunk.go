// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

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
