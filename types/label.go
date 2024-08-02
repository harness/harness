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
	"unicode"
	"unicode/utf8"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"
)

const (
	maxLabelLength = 50
)

type Label struct {
	ID          int64           `json:"id"`
	SpaceID     *int64          `json:"space_id,omitempty"`
	RepoID      *int64          `json:"repo_id,omitempty"`
	Scope       int64           `json:"scope"`
	Key         string          `json:"key"`
	Description string          `json:"description"`
	Type        enum.LabelType  `json:"type"`
	Color       enum.LabelColor `json:"color"`
	ValueCount  int64           `json:"value_count"`
	Created     int64           `json:"created"`
	Updated     int64           `json:"updated"`
	CreatedBy   int64           `json:"created_by"`
	UpdatedBy   int64           `json:"updated_by"`
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
	Values []*LabelValue `json:"values"`
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
	PullReqID  int64           `json:"-"`
	LabelID    int64           `json:"id"`
	LabelKey   string          `json:"key"`
	LabelColor enum.LabelColor `json:"color,omitempty"`
	ValueCount int64           `json:"value_count"`
	Value      *string         `json:"value,omitempty"`
	ValueColor *string         `json:"value_color,omitempty"`
}

type ScopeData struct {
	// Scope = 0 is repo, scope >= 1 is a depth level of a space
	Scope int64       `json:"scope"`
	Space *Space      `json:"space,omitempty"`
	Repo  *Repository `json:"repository,omitempty"`
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
	Inherited bool `json:"inherited,omitempty"`
}

type DefineLabelInput struct {
	Key         string          `json:"key"`
	Type        enum.LabelType  `json:"type"`
	Description string          `json:"description"`
	Color       enum.LabelColor `json:"color"`
}

func (in DefineLabelInput) Validate() error {
	if err := validateLabelText(&in.Key, "key"); err != nil {
		return err
	}

	if err := validateLabelType(in.Type); err != nil {
		return err
	}

	err := validateLabelColor(in.Color)
	if err != nil {
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

func (in UpdateLabelInput) Validate() error {
	if in.Key != nil {
		if err := validateLabelText(in.Key, "key"); err != nil {
			return err
		}
	}

	if in.Type != nil {
		if err := validateLabelType(*in.Type); err != nil {
			return err
		}
	}

	if in.Color != nil {
		err := validateLabelColor(*in.Color)
		if err != nil {
			return err
		}
	}

	return nil
}

type DefineValueInput struct {
	Value string          `json:"value"`
	Color enum.LabelColor `json:"color"`
}

func (in DefineValueInput) Validate() error {
	if err := validateLabelText(&in.Value, "value"); err != nil {
		return err
	}

	if err := validateLabelColor(in.Color); err != nil {
		return err
	}

	return nil
}

type UpdateValueInput struct {
	Value *string          `json:"value"`
	Color *enum.LabelColor `json:"color"`
}

func (in UpdateValueInput) Validate() error {
	if in.Value != nil {
		if err := validateLabelText(in.Value, "value"); err != nil {
			return err
		}
	}

	if in.Color != nil {
		if err := validateLabelColor(*in.Color); err != nil {
			return err
		}
	}

	return nil
}

type PullReqCreateInput struct {
	LabelID int64  `json:"label_id"`
	ValueID *int64 `json:"value_id"`
	Value   string `json:"value"`
}

type PullReqUpdateInput struct {
	LabelValueID *int64 `json:"label_value_id,omitempty"`
}

func (in PullReqCreateInput) Validate() error {
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

func (in *SaveInput) Validate() error {
	if err := in.Label.Validate(); err != nil {
		return err
	}

	for _, value := range in.Values {
		if err := value.Validate(); err != nil {
			return err
		}
	}

	return nil
}

var labelTypes, _ = enum.GetAllLabelTypes()

func validateLabelText(text *string, typ string) error {
	*text = strings.TrimSpace(*text)

	if len(*text) == 0 {
		return errors.InvalidArgument("%s must be a non-empty string", typ)
	}

	if utf8.RuneCountInString(*text) > maxLabelLength {
		return errors.InvalidArgument("%s can have at most %d characters", typ, maxLabelLength)
	}

	for _, ch := range *text {
		if unicode.IsControl(ch) {
			return errors.InvalidArgument("%s cannot contain control characters", typ)
		}
	}

	return nil
}

func validateLabelType(typ enum.LabelType) error {
	if _, ok := typ.Sanitize(); !ok {
		return errors.InvalidArgument("label type must be in %v", labelTypes)
	}
	return nil
}

var colorTypes, _ = enum.GetAllLabelColors()

func validateLabelColor(color enum.LabelColor) error {
	_, ok := color.Sanitize()
	if !ok {
		return errors.InvalidArgument("color type must be in %v", colorTypes)
	}

	return nil
}
