// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
