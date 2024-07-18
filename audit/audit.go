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

package audit

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/types"
)

var (
	ErrActionUndefined              = errors.New("undefined action")
	ErrResourceTypeUndefined        = errors.New("undefined resource type")
	ErrResourceIdentifierIsRequired = errors.New("resource identifier is required")
	ErrUserIsRequired               = errors.New("user is required")
	ErrSpacePathIsRequired          = errors.New("space path is required")
)

const (
	RepoName = "repoName"
)

type Action string

const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated" // update default branch, switching default branch, updating description
	ActionDeleted Action = "deleted"
)

func (a Action) Validate() error {
	switch a {
	case ActionCreated, ActionUpdated, ActionDeleted:
		return nil
	default:
		return ErrActionUndefined
	}
}

type ResourceType string

const (
	ResourceTypeRepository         ResourceType = "repository"
	ResourceTypeBranchRule         ResourceType = "branch_rule"
	ResourceTypeRepositorySettings ResourceType = "repository_settings"
)

func (a ResourceType) Validate() error {
	switch a {
	case ResourceTypeRepository,
		ResourceTypeBranchRule,
		ResourceTypeRepositorySettings:
		return nil
	default:
		return ErrResourceTypeUndefined
	}
}

type Resource struct {
	Type       ResourceType
	Identifier string
	Data       map[string]string
}

func NewResource(rtype ResourceType, identifier string, keyValues ...string) Resource {
	r := Resource{
		Type:       rtype,
		Identifier: identifier,
		Data:       make(map[string]string, len(keyValues)),
	}
	for i := 0; i < len(keyValues); i += 2 {
		k, v := keyValues[i], keyValues[i+1]
		r.Data[k] = v
	}
	return r
}

func (r Resource) Validate() error {
	if err := r.Type.Validate(); err != nil {
		return err
	}
	if r.Identifier == "" {
		return ErrResourceIdentifierIsRequired
	}
	return nil
}

func (r Resource) DataAsSlice() []string {
	slice := make([]string, 0, len(r.Data)*2)
	for k, v := range r.Data {
		slice = append(slice, k, v)
	}
	return slice
}

type DiffObject struct {
	OldObject any
	NewObject any
}

type Event struct {
	ID            string
	Timestamp     int64
	Action        Action          // example: ActionCreated
	User          types.Principal // example: Admin
	SpacePath     string          // example: /root/projects
	Resource      Resource
	DiffObject    DiffObject
	ClientIP      string
	RequestMethod string
	Data          map[string]string // internal data like correlationID/requestID
}

func (e *Event) Validate() error {
	if err := e.Action.Validate(); err != nil {
		return fmt.Errorf("invalid action: %w", err)
	}
	if e.User.UID == "" {
		return ErrUserIsRequired
	}
	if e.SpacePath == "" {
		return ErrSpacePathIsRequired
	}
	if err := e.Resource.Validate(); err != nil {
		return fmt.Errorf("invalid resource: %w", err)
	}
	return nil
}

type Noop struct{}

func New() *Noop {
	return &Noop{}
}

func (s *Noop) Log(
	context.Context,
	types.Principal,
	Resource,
	Action,
	string,
	...Option,
) error {
	// No implementation
	return nil
}

type FuncOption func(e *Event)

func (f FuncOption) Apply(event *Event) {
	f(event)
}

type Option interface {
	Apply(e *Event)
}

func WithID(value string) FuncOption {
	return func(e *Event) {
		e.ID = value
	}
}

func WithNewObject(value any) FuncOption {
	return func(e *Event) {
		e.DiffObject.NewObject = value
	}
}

func WithOldObject(value any) FuncOption {
	return func(e *Event) {
		e.DiffObject.OldObject = value
	}
}

func WithClientIP(value string) FuncOption {
	return func(e *Event) {
		e.ClientIP = value
	}
}

func WithRequestMethod(value string) FuncOption {
	return func(e *Event) {
		e.RequestMethod = value
	}
}

func WithData(keyValues ...string) FuncOption {
	return func(e *Event) {
		if e.Data == nil {
			e.Data = make(map[string]string)
		}
		for i := 0; i < len(keyValues); i += 2 {
			k, v := keyValues[i], keyValues[i+1]
			e.Data[k] = v
		}
	}
}
