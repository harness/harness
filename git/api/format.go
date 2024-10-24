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

package api

const (
	fmtEOL  = "%n"
	fmtPerc = "%%"
	fmtZero = "%x00"

	fmtCommitHash   = "%H"
	fmtTreeHash     = "%T"
	fmtParentHashes = "%P"

	fmtAuthorName  = "%an"
	fmtAuthorEmail = "%ae"
	fmtAuthorTime  = "%aI" // ISO 8601
	fmtAuthorUnix  = "%at" // Unix timestamp

	fmtCommitterName  = "%cn"
	fmtCommitterEmail = "%ce"
	fmtCommitterTime  = "%cI" // ISO 8601
	fmtCommitterUnix  = "%ct" // Unix timestamp

	fmtSubject = "%s"
	fmtMessage = "%B"

	fmtFieldObjectType = "%(objecttype)"
	fmtFieldPath       = "%(path)"
)
