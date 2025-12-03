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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type CodeOwnerEvaluation struct {
	EvaluationEntries []CodeOwnerEvaluationEntry `json:"evaluation_entries"`
	FileSha           string                     `json:"file_sha"`
}

type CodeOwnerEvaluationEntry struct {
	LineNumber                int64                      `json:"line_number"`
	Pattern                   string                     `json:"pattern"`
	OwnerEvaluations          []OwnerEvaluation          `json:"owner_evaluations"`
	UserGroupOwnerEvaluations []UserGroupOwnerEvaluation `json:"user_group_owner_evaluations"`
}

type UserGroupOwnerEvaluation struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Evaluations []OwnerEvaluation `json:"evaluations"`
}

type OwnerEvaluation struct {
	Owner          PrincipalInfo              `json:"owner"`
	ReviewDecision enum.PullReqReviewDecision `json:"review_decision"`
	ReviewSHA      string                     `json:"review_sha"`
}

type CodeOwnersValidation struct {
	Violations []CodeOwnersViolation `json:"violations"`
}

type CodeOwnersViolation struct {
	Code    enum.CodeOwnerViolationCode `json:"code"`
	Message string                      `json:"message"`
	Params  []any                       `json:"params"`
}

func (violations *CodeOwnersValidation) Add(code enum.CodeOwnerViolationCode, message string) {
	violations.Violations = append(violations.Violations, CodeOwnersViolation{
		Code:    code,
		Message: message,
		Params:  nil,
	})
}

func (violations *CodeOwnersValidation) Addf(code enum.CodeOwnerViolationCode, format string, params ...any) {
	violations.Violations = append(violations.Violations, CodeOwnersViolation{
		Code:    code,
		Message: fmt.Sprintf(format, params...),
		Params:  params,
	})
}
