// Copyright 2026 Harness, Inc.
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

// ContentViolation is a single problem found while validating the content of a file.
type ContentViolation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Params  []any  `json:"params,omitempty"`
	// Line is the 1-based line the violation refers to; omitted when there is no meaningful line.
	Line int64 `json:"line,omitempty"`
}

// ContentValidation holds the violations found in the content of a single file.
type ContentValidation struct {
	Path       string             `json:"path"`
	Violations []ContentViolation `json:"violations"`
}
