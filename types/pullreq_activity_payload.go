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
	MergeMethod   enum.MergeMethod `json:"merge_method"`
	MergeSHA      string           `json:"merge_sha"`
	TargetSHA     string           `json:"target_sha"`
	SourceSHA     string           `json:"source_sha"`
	RulesBypassed bool             `json:"rules_bypassed,omitempty"`
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

type PullRequestActivityPayloadReviewerDelete struct {
	CommitSHA   string                     `json:"commit_sha"`
	Decision    enum.PullReqReviewDecision `json:"decision"`
	PrincipalID int64                      `json:"principal_id"`
}

func (a *PullRequestActivityPayloadReviewerDelete) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeReviewerDelete
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

type PullRequestActivityLabel struct {
	Label         string                        `json:"label"`
	LabelColor    enum.LabelColor               `json:"label_color"`
	Value         *string                       `json:"value,omitempty"`
	ValueColor    *enum.LabelColor              `json:"value_color,omitempty"`
	OldValue      *string                       `json:"old_value,omitempty"`
	OldValueColor *enum.LabelColor              `json:"old_value_color,omitempty"`
	Type          enum.PullReqLabelActivityType `json:"type"`
}

func (a *PullRequestActivityLabel) ActivityType() enum.PullReqActivityType {
	return enum.PullReqActivityTypeLabelModify
}
