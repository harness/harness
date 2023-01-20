// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	Updated   int64  `json:"-"` // not returned, it's updated by the server internally. Clients should use EditedAt.
	Edited    int64  `json:"edited"`
	Deleted   *int64 `json:"deleted"`

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
	Resolved   *int64 `json:"resolved"`

	Author   PrincipalInfo  `json:"author"`
	Resolver *PrincipalInfo `json:"resolver"`
}

func (a *PullReqActivity) IsReplyable() bool {
	return (a.Type == enum.PullReqActivityTypeComment || a.Type == enum.PullReqActivityTypeCodeComment) &&
		a.SubOrder == 0
}

func (a *PullReqActivity) IsReply() bool {
	return a.SubOrder > 0
}

// SetPayload sets the payload and verifies it's of correct type for the activity.
func (a *PullReqActivity) SetPayload(payload pullReqActivityPayload) error {
	if payload == nil {
		a.PayloadRaw = json.RawMessage(nil)
		return nil
	}

	if payload.activityType() != a.Type {
		return fmt.Errorf("wrong payload type %T for activity %s, payload is for %s",
			payload, a.Type, payload.activityType())
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
func (a *PullReqActivity) GetPayload() (pullReqActivityPayload, error) {
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

// pullReqActivityPayload is an interface used to identify PR activity payload types.
// The approach is inspired by what protobuf is doing for oneof.
type pullReqActivityPayload interface {
	// pullReqActivityType returns the pr activity type the payload is meant for.
	// NOTE: this allows us to do easy payload type verification without any kind of reflection.
	activityType() enum.PullReqActivityType
}

// activityPayloadFactoryMethod is an alias for a function that creates a new pullReqActivityPayload.
// NOTE: this is used to create new instances for activities on the fly (to avoid reflection)
// NOTE: we could add new() to pullReqActivityPayload interface, but it shouldn't be the payloads' responsibility.
type activityPayloadFactoryMethod func() pullReqActivityPayload

// allPullReqActivityPayloads is a map that contains the payload factory methods for all activity types with payload.
var allPullReqActivityPayloads = func(
	factoryMethods []activityPayloadFactoryMethod) map[enum.PullReqActivityType]activityPayloadFactoryMethod {
	payloadMap := make(map[enum.PullReqActivityType]activityPayloadFactoryMethod)
	for _, factoryMethod := range factoryMethods {
		payloadMap[factoryMethod().activityType()] = factoryMethod
	}
	return payloadMap
}([]activityPayloadFactoryMethod{
	func() pullReqActivityPayload { return &PullRequestActivityPayloadComment{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadMerge{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadStateChange{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadTitleChange{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadReviewSubmit{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadBranchUpdate{} },
	func() pullReqActivityPayload { return &PullRequestActivityPayloadBranchDelete{} },
})

// newPayloadForActivity returns a new payload instance for the requested activity type.
func newPayloadForActivity(t enum.PullReqActivityType) (pullReqActivityPayload, error) {
	payloadFactoryMethod, ok := allPullReqActivityPayloads[t]
	if !ok {
		return nil, fmt.Errorf("pr activity type '%s' doesn't have a payload", t)
	}

	return payloadFactoryMethod(), nil
}

// PullRequestActivityPayloadComment represents the payload for a comment.
// NOTE: Allow UI to store whatever needed for code comments until we have a proper solution.
type PullRequestActivityPayloadComment map[string]interface{}

func (a *PullRequestActivityPayloadComment) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeComment
}

type PullRequestActivityPayloadMerge struct {
	MergeMethod enum.MergeMethod `json:"merge_method"`
	SHA         string           `json:"sha"`
}

func (a *PullRequestActivityPayloadMerge) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeMerge
}

type PullRequestActivityPayloadStateChange struct {
	Old     enum.PullReqState `json:"old"`
	New     enum.PullReqState `json:"new"`
	IsDraft bool              `json:"is_draft"`
	Message string            `json:"message,omitempty"`
}

func (a *PullRequestActivityPayloadStateChange) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeStateChange
}

type PullRequestActivityPayloadTitleChange struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (a *PullRequestActivityPayloadTitleChange) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeTitleChange
}

type PullRequestActivityPayloadReviewSubmit struct {
	Message  string                     `json:"message,omitempty"`
	Decision enum.PullReqReviewDecision `json:"decision"`
}

func (a *PullRequestActivityPayloadReviewSubmit) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeReviewSubmit
}

type PullRequestActivityPayloadBranchUpdate struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (a *PullRequestActivityPayloadBranchUpdate) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeBranchUpdate
}

type PullRequestActivityPayloadBranchDelete struct {
	SHA string `json:"sha"`
}

func (a *PullRequestActivityPayloadBranchDelete) activityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeBranchDelete
}
