// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gogs

import "time"

type Milestone struct {
	ID           int64      `json:"id"`
	State        string     `json:"state"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	Closed       *time.Time `json:"closed_at"`
	Deadline     *time.Time `json:"due_on"`
}
