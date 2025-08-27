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

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types/enum"
)

// PullReqActivity represents a pull request activity.
type PullReqActivity struct {
	ID      int64 `json:"id"`
	Version int64 `json:"-"` // not returned, it's an internal field

	CreatedBy int64  `json:"-"` // not returned, because the author info is in the Author field
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"` // we need updated to determine the latest version reliably.
	Edited    int64  `json:"edited"`
	Deleted   *int64 `json:"deleted,omitempty"`

	ParentID  *int64 `json:"parent_id"`
	RepoID    int64  `json:"repo_id"`
	PullReqID int64  `json:"pullreq_id"`

	Order    int64 `json:"order"`
	SubOrder int64 `json:"sub_order"`
	ReplySeq int64 `json:"-"` // not returned, because it's a server's internal field

	Type enum.PullReqActivityType `json:"type"`
	Kind enum.PullReqActivityKind `json:"kind"`

	Text       string                   `json:"text"`
	PayloadRaw json.RawMessage          `json:"payload"`
	Metadata   *PullReqActivityMetadata `json:"metadata,omitempty"`

	ResolvedBy *int64 `json:"-"` // not returned, because the resolver info is in the Resolver field
	Resolved   *int64 `json:"resolved,omitempty"`

	Author   PrincipalInfo  `json:"author"`
	Resolver *PrincipalInfo `json:"resolver,omitempty"`

	CodeComment *CodeCommentFields `json:"code_comment,omitempty"`

	// used only in response
	Mentions      map[int64]*PrincipalInfo `json:"mentions,omitempty"`
	GroupMentions map[int64]*UserGroupInfo `json:"user_group_mentions,omitempty"`
}

func (a *PullReqActivity) IsValidCodeComment() bool {
	return a.Type == enum.PullReqActivityTypeCodeComment &&
		a.Kind == enum.PullReqActivityKindChangeComment &&
		a.CodeComment != nil
}

func (a *PullReqActivity) AsCodeComment() *CodeComment {
	if !a.IsValidCodeComment() {
		return &CodeComment{}
	}
	return &CodeComment{
		ID:      a.ID,
		Version: a.Version,
		Updated: a.Updated,
		CodeCommentFields: CodeCommentFields{
			Outdated:     a.CodeComment.Outdated,
			MergeBaseSHA: a.CodeComment.MergeBaseSHA,
			SourceSHA:    a.CodeComment.SourceSHA,
			Path:         a.CodeComment.Path,
			LineNew:      a.CodeComment.LineNew,
			SpanNew:      a.CodeComment.SpanNew,
			LineOld:      a.CodeComment.LineOld,
			SpanOld:      a.CodeComment.SpanOld,
		},
	}
}

func (a *PullReqActivity) IsReplyable() bool {
	return (a.Type == enum.PullReqActivityTypeComment || a.Type == enum.PullReqActivityTypeCodeComment) &&
		a.SubOrder == 0
}

func (a *PullReqActivity) IsReply() bool {
	return a.SubOrder > 0
}

// IsBlocking returns true if the pull request activity (comment/code-comment) is blocking the pull request merge.
func (a *PullReqActivity) IsBlocking() bool {
	return a.SubOrder == 0 && a.Resolved == nil && a.Deleted == nil && a.Kind != enum.PullReqActivityKindSystem
}

// SetPayload sets the payload and verifies it's of correct type for the activity.
func (a *PullReqActivity) SetPayload(payload PullReqActivityPayload) error {
	if payload == nil {
		a.PayloadRaw = json.RawMessage(nil)
		return nil
	}

	if payload.ActivityType() != a.Type {
		return fmt.Errorf("wrong payload type %T for activity %s, payload is for %s",
			payload, a.Type, payload.ActivityType())
	}

	var err error
	if a.PayloadRaw, err = json.Marshal(payload); err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return nil
}

// GetPayload returns the payload of the activity.
// An error is returned in case there's an issue retrieving the payload from its raw value.
// NOTE: To ensure rawValue gets changed always use SetPayload() with the updated payload.
func (a *PullReqActivity) GetPayload() (PullReqActivityPayload, error) {
	// jsonMessage could also contain "null" - we still want to return ErrNoPayload in that case
	if a.PayloadRaw == nil ||
		bytes.Equal(a.PayloadRaw, jsonRawMessageNullBytes) {
		return nil, ErrNoPayload
	}

	payload, err := newPayloadForActivity(a.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create new payload: %w", err)
	}

	if err = json.Unmarshal(a.PayloadRaw, payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return payload, nil
}

// UpdateMetadata updates the metadata with the provided options.
func (a *PullReqActivity) UpdateMetadata(updates ...PullReqActivityMetadataUpdate) {
	if a.Metadata == nil {
		a.Metadata = &PullReqActivityMetadata{}
	}

	for _, update := range updates {
		update.apply(a.Metadata)
	}

	if a.Metadata.IsEmpty() {
		a.Metadata = nil
	}
}

// PullReqActivityFilter stores pull request activity query parameters.
type PullReqActivityFilter struct {
	After  int64 `json:"after"`
	Before int64 `json:"before"`
	Limit  int   `json:"limit"`

	Types []enum.PullReqActivityType `json:"type"`
	Kinds []enum.PullReqActivityKind `json:"kind"`
}
