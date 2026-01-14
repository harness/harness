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

// Diff file header extensions. From: https://git-scm.com/docs/git-diff#generate_patch_text_with_p
const (
	DiffExtHeaderOldMode         = "old mode"            // old mode <mode>
	DiffExtHeaderNewMode         = "new mode"            // new mode <mode>
	DiffExtHeaderDeletedFileMode = "deleted file mode"   // deleted file mode <mode>
	DiffExtHeaderNewFileMode     = "new file mode"       // new file mode <mode>
	DiffExtHeaderCopyFrom        = "copy from"           // copy from <path>
	DiffExtHeaderCopyTo          = "copy to"             // copy to <path>
	DiffExtHeaderRenameFrom      = "rename from"         // rename from <path>
	DiffExtHeaderRenameTo        = "rename to"           // rename to <path>
	DiffExtHeaderSimilarity      = "similarity index"    // similarity index <number>
	DiffExtHeaderDissimilarity   = "dissimilarity index" // dissimilarity index <number>
	DiffExtHeaderIndex           = "index"               // index <hash>..<hash> <mode>
)
