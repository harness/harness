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

package webhook

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Preprocessor interface {
	PreprocessCreateInput(enum.PrincipalType, *types.WebhookCreateInput) (bool, error)
	PreprocessUpdateInput(enum.PrincipalType, *types.WebhookUpdateInput) (bool, error)
	PreprocessFilter(enum.PrincipalType, *types.WebhookFilter)
	IsInternalCall(enum.PrincipalType) bool
}

type NoopPreprocessor struct {
}

// PreprocessCreateInput always return false for internal.
func (p NoopPreprocessor) PreprocessCreateInput(
	enum.PrincipalType,
	*types.WebhookCreateInput,
) (bool, error) {
	return false, nil
}

// PreprocessUpdateInput always return false for internal.
func (p NoopPreprocessor) PreprocessUpdateInput(
	enum.PrincipalType,
	*types.WebhookUpdateInput,
) (bool, error) {
	return false, nil
}

func (p NoopPreprocessor) PreprocessFilter(_ enum.PrincipalType, filter *types.WebhookFilter) {
	if filter.Order == enum.OrderDefault {
		filter.Order = enum.OrderAsc
	}

	// always skip internal for requests from handler
	filter.SkipInternal = true
}

func (p NoopPreprocessor) IsInternalCall(enum.PrincipalType) bool {
	return false
}
