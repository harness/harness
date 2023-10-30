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

package enum

type CodeOwnerViolationCode string

const (
	// CodeOwnerViolationCodeUserNotFound occurs when user in codeowners file is not present.
	CodeOwnerViolationCodeUserNotFound CodeOwnerViolationCode = "user_not_found"
	// CodeOwnerViolationCodePatternInvalid occurs when a pattern in codeowners file is incorrect.
	CodeOwnerViolationCodePatternInvalid CodeOwnerViolationCode = "pattern_invalid"
	// CodeOwnerViolationCodePatternEmpty occurs when a pattern in codeowners file is empty.
	CodeOwnerViolationCodePatternEmpty CodeOwnerViolationCode = "pattern_empty"
)

func (CodeOwnerViolationCode) Enum() []interface{} { return toInterfaceSlice(codeOwnerViolationCodes) }

var codeOwnerViolationCodes = sortEnum([]CodeOwnerViolationCode{
	CodeOwnerViolationCodeUserNotFound,
	CodeOwnerViolationCodePatternInvalid,
	CodeOwnerViolationCodePatternEmpty,
})
