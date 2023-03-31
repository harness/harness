// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type CodeComment struct {
	ID      int64 `db:"pullreq_activity_id"`
	Version int64 `db:"pullreq_activity_version"`
	Updated int64 `db:"pullreq_activity_updated"`

	Outdated     bool   `db:"pullreq_activity_outdated"`
	MergeBaseSHA string `db:"pullreq_activity_code_comment_merge_base_sha"`
	SourceSHA    string `db:"pullreq_activity_code_comment_source_sha"`
	Path         string `db:"pullreq_activity_code_comment_path"`
	LineNew      int    `db:"pullreq_activity_code_comment_line_new"`
	SpanNew      int    `db:"pullreq_activity_code_comment_span_new"`
	LineOld      int    `db:"pullreq_activity_code_comment_line_old"`
	SpanOld      int    `db:"pullreq_activity_code_comment_span_old"`
}
