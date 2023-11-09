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
	"errors"
	"fmt"

	"github.com/harness/gitness/types/enum"
)

var (
	// jsonRawMessageNullBytes represents the byte array that's equivalent to a nil json.RawMessage.
	jsonRawMessageNullBytes = []byte("null")

	// ErrNoPayload is returned in case the activity doesn't have any payload set.
	ErrNoPayload = errors.New("activity has no payload")
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

	Text       string                 `json:"text"`
	PayloadRaw json.RawMessage        `json:"payload"`
	Metadata   map[string]interface{} `json:"metadata"`

	ResolvedBy *int64 `json:"-"` // not returned, because the resolver info is in the Resolver field
	Resolved   *int64 `json:"resolved,omitempty"`

	Author   PrincipalInfo  `json:"author"`
	Resolver *PrincipalInfo `json:"resolver,omitempty"`

	CodeComment *CodeCommentFields `json:"code_comment,omitempty"`
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

// PullReqActivityFilter stores pull request activity query parameters.
type PullReqActivityFilter struct {
	After  int64 `json:"after"`
	Before int64 `json:"before"`
	Limit  int   `json:"limit"`

	Types []enum.PullReqActivityType `json:"type"`
	Kinds []enum.PullReqActivityKind `json:"kind"`
}

// PullReqActivityPayload is an interface used to identify PR activity payload types.
// The approach is inspired by what protobuf is doing for oneof.
type PullReqActivityPayload interface {
	// ActivityType returns the pr activity type the payload is meant for.
	// NOTE: this allows us to do easy payload type verification without any kind of reflection.
	ActivityType() enum.PullReqActivityType
}

// activityPayloadFactoryMethod is an alias for a function that creates a new PullReqActivityPayload.
// NOTE: this is used to create new instances for activities on the fly (to avoid reflection)
// NOTE: we could add new() to PullReqActivityPayload interface, but it shouldn't be the payloads' responsibility.
type activityPayloadFactoryMethod func() PullReqActivityPayload

// allPullReqActivityPayloads is a map that contains the payload factory methods for all activity types with payload.
var allPullReqActivityPayloads = func(
	factoryMethods []activityPayloadFactoryMethod,
) map[enum.PullReqActivityType]activityPayloadFactoryMethod {
	payloadMap := make(map[enum.PullReqActivityType]activityPayloadFactoryMethod)
	for _, factoryMethod := range factoryMethods {
		payloadMap[factoryMethod().ActivityType()] = factoryMethod
	}
	return payloadMap
}([]activityPayloadFactoryMethod{
	func() PullReqActivityPayload { return PullRequestActivityPayloadComment{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadCodeComment{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadMerge{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadStateChange{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadTitleChange{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadReviewSubmit{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadBranchUpdate{} },
	func() PullReqActivityPayload { return &PullRequestActivityPayloadBranchDelete{} },
})

// newPayloadForActivity returns a new payload instance for the requested activity type.
func newPayloadForActivity(t enum.PullReqActivityType) (PullReqActivityPayload, error) {
	payloadFactoryMethod, ok := allPullReqActivityPayloads[t]
	if !ok {
		return nil, fmt.Errorf("pr activity type '%s' doesn't have a payload", t)
	}

	return payloadFactoryMethod(), nil
}

type PullRequestActivityPayloadComment struct{}

func (a PullRequestActivityPayloadComment) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeComment
}

type PullRequestActivityPayloadCodeComment struct {
	Title        string   `json:"title"`
	Lines        []string `json:"lines"`
	LineStartNew bool     `json:"line_start_new"`
	LineEndNew   bool     `json:"line_end_new"`
}

func (a *PullRequestActivityPayloadCodeComment) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeCodeComment
}

type PullRequestActivityPayloadMerge struct {
	MergeMethod enum.MergeMethod `json:"merge_method"`
	MergeSHA    string           `json:"merge_sha"`
	TargetSHA   string           `json:"target_sha"`
	SourceSHA   string           `json:"source_sha"`
}

func (a *PullRequestActivityPayloadMerge) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeMerge
}

type PullRequestActivityPayloadStateChange struct {
	Old      enum.PullReqState `json:"old"`
	New      enum.PullReqState `json:"new"`
	OldDraft bool              `json:"old_draft"`
	NewDraft bool              `json:"new_draft"`
}

func (a *PullRequestActivityPayloadStateChange) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeStateChange
}

type PullRequestActivityPayloadTitleChange struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (a *PullRequestActivityPayloadTitleChange) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeTitleChange
}

type PullRequestActivityPayloadReviewSubmit struct {
	CommitSHA string                     `json:"commit_sha"`
	Decision  enum.PullReqReviewDecision `json:"decision"`
}

func (a *PullRequestActivityPayloadReviewSubmit) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeReviewSubmit
}

type PullRequestActivityPayloadBranchUpdate struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (a *PullRequestActivityPayloadBranchUpdate) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeBranchUpdate
}

type PullRequestActivityPayloadBranchDelete struct {
	SHA string `json:"sha"`
}

func (a *PullRequestActivityPayloadBranchDelete) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeBranchDelete
}
