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

package githook

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
)

type PreReceiveExtender interface {
	Extend(
		context.Context,
		RestrictedGIT,
		*auth.Session,
		*types.RepositoryCore,
		types.GithookPreReceiveInput,
		*hook.Output,
	) error
}

type UpdateExtender interface {
	Extend(
		context.Context,
		RestrictedGIT,
		*auth.Session,
		*types.RepositoryCore,
		types.GithookUpdateInput,
		*hook.Output,
	) error
}

type PostReceiveExtender interface {
	Extend(
		context.Context,
		RestrictedGIT,
		*auth.Session,
		*types.RepositoryCore,
		types.GithookPostReceiveInput,
		*hook.Output,
	) error
}

type NoOpPreReceiveExtender struct {
}

func NewPreReceiveExtender() PreReceiveExtender {
	return NoOpPreReceiveExtender{}
}

func (NoOpPreReceiveExtender) Extend(
	context.Context,
	RestrictedGIT,
	*auth.Session,
	*types.RepositoryCore,
	types.GithookPreReceiveInput,
	*hook.Output,
) error {
	return nil
}

type NoOpUpdateExtender struct {
}

func NewUpdateExtender() UpdateExtender {
	return NoOpUpdateExtender{}
}

func (NoOpUpdateExtender) Extend(
	context.Context,
	RestrictedGIT,
	*auth.Session,
	*types.RepositoryCore,
	types.GithookUpdateInput,
	*hook.Output,
) error {
	return nil
}

type NoOpPostReceiveExtender struct {
}

func NewPostReceiveExtender() PostReceiveExtender {
	return NoOpPostReceiveExtender{}
}

func (NoOpPostReceiveExtender) Extend(
	context.Context,
	RestrictedGIT,
	*auth.Session,
	*types.RepositoryCore,
	types.GithookPostReceiveInput,
	*hook.Output,
) error {
	return nil
}
