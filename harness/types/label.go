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
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"
)

type Label struct {
	ID           int64           `json:"id"`
	SpaceID      *int64          `json:"space_id,omitempty"`
	RepoID       *int64          `json:"repo_id,omitempty"`
	Scope        int64           `json:"scope"`
	Key          string          `json:"key"`
	Description  string          `json:"description"`
	Type         enum.LabelType  `json:"type"`
	Color        enum.LabelColor `json:"color"`
	ValueCount   int64           `json:"value_count"`
	Created      int64           `json:"created"`
	Updated      int64           `json:"updated"`
	CreatedBy    int64           `json:"created_by"`
	UpdatedBy    int64           `json:"updated_by"`
	PullreqCount int64           `json:"pullreq_count,omitempty"`
}

type LabelValue struct {
	ID        int64           `json:"id"`
	LabelID   int64           `json:"label_id"`
	Value     string          `json:"value"`
	Color     enum.LabelColor `json:"color"`
	Created   int64           `json:"created"`
	Updated   int64           `json:"updated"`
	CreatedBy int64           `json:"created_by"`
	UpdatedBy int64           `json:"updated_by"`
}

type LabelWithValues struct {
	Label  `json:"label"`
	Values []*LabelValue `json:"values,omitempty"`
}

// Used to assign label to pullreq.
type PullReqLabel struct {
	PullReqID int64  `json:"pullreq_id"`
	LabelID   int64  `json:"label_id"`
	ValueID   *int64 `json:"value_id,omitempty"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	CreatedBy int64  `json:"created_by"`
	UpdatedBy int64  `json:"updated_by"`
}

type LabelInfo struct {
	SpaceID  *int64          `json:"-"`
	RepoID   *int64          `json:"-"`
	Scope    int64           `json:"scope"`
	ID       int64           `json:"id"`
	Type     enum.LabelType  `json:"type"`
	Key      string          `json:"key"`
	Color    enum.LabelColor `json:"color"`
	Assigned *bool           `json:"assigned,omitempty"`
}

type LabelValueInfo struct {
	LabelID *int64  `json:"-"`
	ID      *int64  `json:"id,omitempty"`
	Value   *string `json:"value,omitempty"`
	Color   *string `json:"color,omitempty"`
}

type LabelAssignment struct {
	LabelInfo
	AssignedValue *LabelValueInfo   `json:"assigned_value,omitempty"`
	Values        []*LabelValueInfo `json:"values,omitempty"` // query param ?assignable=true
}

type LabelPullReqAssignmentInfo struct {
	PullReqID  int64            `json:"-"`
	LabelID    int64            `json:"id"`
	LabelKey   string           `json:"key"`
	LabelColor enum.LabelColor  `json:"color,omitempty"`
	LabelScope int64            `json:"scope"`
	ValueCount int64            `json:"value_count"`
	ValueID    *int64           `json:"value_id,omitempty"`
	Value      *string          `json:"value,omitempty"`
	ValueColor *enum.LabelColor `json:"value_color,omitempty"`
}

type ScopeData struct {
	// Scope = 0 is repo, scope >= 1 is a depth level of a space
	Scope int64           `json:"scope"`
	Space *SpaceCore      `json:"space,omitempty"`
	Repo  *RepositoryCore `json:"repository,omitempty"`
}

// Used to fetch label and values from a repo and space hierarchy.
type ScopesLabels struct {
	ScopeData []*ScopeData       `json:"scope_data"`
	LabelData []*LabelAssignment `json:"label_data"`
}

// LabelFilter stores label query parameters.
type AssignableLabelFilter struct {
	ListQueryFilter
	Assignable bool `json:"assignable,omitempty"`
}
type LabelFilter struct {
	ListQueryFilter
	Inherited           bool `json:"inherited,omitempty"`
	IncludePullreqCount bool `json:"pullreq_count,omitempty"`
}

type DefineLabelInput struct {
	Key         string          `json:"key"`
	Type        enum.LabelType  `json:"type"`
	Description string          `json:"description"`
	Color       enum.LabelColor `json:"color"`
}

func (in *DefineLabelInput) Sanitize() error {
	if err := SanitizeTag(&in.Key, TagPartTypeKey, true); err != nil {
		return err
	}

	sanitizeDescription(&in.Description)

	if err := sanitizeLabelType(&in.Type); err != nil {
		return err
	}

	if err := sanitizeLabelColor(&in.Color); err != nil {
		return err
	}

	return nil
}

type UpdateLabelInput struct {
	Key         *string          `json:"key,omitempty"`
	Type        *enum.LabelType  `json:"type,omitempty"`
	Description *string          `json:"description,omitempty"`
	Color       *enum.LabelColor `json:"color,omitempty"`
}

func (in *UpdateLabelInput) Sanitize() error {
	if err := SanitizeTag(in.Key, TagPartTypeKey, true); err != nil {
		return err
	}

	sanitizeDescription(in.Description)

	if err := sanitizeLabelType(in.Type); err != nil {
		return err
	}

	if err := sanitizeLabelColor(in.Color); err != nil {
		return err
	}

	return nil
}

type DefineValueInput struct {
	Value string          `json:"value"`
	Color enum.LabelColor `json:"color"`
}

func (in *DefineValueInput) Sanitize() error {
	if err := SanitizeTag(&in.Value, TagPartTypeValue, true); err != nil {
		return err
	}

	if err := sanitizeLabelColor(&in.Color); err != nil {
		return err
	}

	return nil
}

type UpdateValueInput struct {
	Value *string          `json:"value"`
	Color *enum.LabelColor `json:"color"`
}

func (in *UpdateValueInput) Sanitize() error {
	if in.Value != nil {
		if err := SanitizeTag(in.Value, TagPartTypeValue, true); err != nil {
			return err
		}
	}

	if err := sanitizeLabelColor(in.Color); err != nil {
		return err
	}

	return nil
}

type PullReqLabelAssignInput struct {
	LabelID int64  `json:"label_id"`
	ValueID *int64 `json:"value_id"`
	Value   string `json:"value"`
}

type PullReqUpdateInput struct {
	LabelValueID *int64 `json:"label_value_id,omitempty"`
}

func (in PullReqLabelAssignInput) Validate() error {
	if (in.ValueID != nil && *in.ValueID > 0) && in.Value != "" {
		return errors.InvalidArgument("cannot accept both value id and value")
	}
	return nil
}

type SaveLabelInput struct {
	ID int64 `json:"id"`
	DefineLabelInput
}
type SaveLabelValueInput struct {
	ID int64 `json:"id"`
	DefineValueInput
}

type SaveInput struct {
	Label  SaveLabelInput         `json:"label"`
	Values []*SaveLabelValueInput `json:"values,omitempty"`
}

func (in *SaveInput) Sanitize() error {
	if err := in.Label.Sanitize(); err != nil {
		return err
	}

	for _, value := range in.Values {
		if err := value.Sanitize(); err != nil {
			return err
		}
	}

	return nil
}

func sanitizeDescription(description *string) {
	if description == nil {
		return
	}

	*description = strings.TrimSpace(*description)
}

func sanitizeLabelType(typ *enum.LabelType) error {
	if typ == nil {
		return nil
	}

	*typ = enum.LabelType(trimLowerText(string(*typ)))

	var ok bool
	if *typ, ok = typ.Sanitize(); !ok {
		return errors.InvalidArgument("invalid label type")
	}

	return nil
}

func sanitizeLabelColor(color *enum.LabelColor) error {
	if color == nil {
		return nil
	}

	*color = enum.LabelColor(trimLowerText(string(*color)))

	var ok bool
	if *color, ok = color.Sanitize(); !ok {
		return errors.InvalidArgument("invalid label color")
	}

	return nil
}

func trimLowerText(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}
