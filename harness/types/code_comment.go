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

package types

type CodeComment struct {
	ID      int64 `db:"pullreq_activity_id"`
	Version int64 `db:"pullreq_activity_version"`
	Updated int64 `db:"pullreq_activity_updated"`

	CodeCommentFields
}

type CodeCommentFields struct {
	Outdated     bool   `db:"pullreq_activity_outdated" json:"outdated"`
	MergeBaseSHA string `db:"pullreq_activity_code_comment_merge_base_sha" json:"merge_base_sha"`
	SourceSHA    string `db:"pullreq_activity_code_comment_source_sha" json:"source_sha"`
	Path         string `db:"pullreq_activity_code_comment_path" json:"path"`
	LineNew      int    `db:"pullreq_activity_code_comment_line_new" json:"line_new"`
	SpanNew      int    `db:"pullreq_activity_code_comment_span_new" json:"span_new"`
	LineOld      int    `db:"pullreq_activity_code_comment_line_old" json:"line_old"`
	SpanOld      int    `db:"pullreq_activity_code_comment_span_old" json:"span_old"`
}
