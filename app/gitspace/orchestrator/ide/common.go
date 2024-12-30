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

package ide

import (
	"fmt"

	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
)

func getIDEDownloadURL(
	args map[gitspaceTypes.IDEArg]interface{},
) (types.IntellijDownloadURL, error) {
	downloadURL, exists := args[gitspaceTypes.IDEDownloadURLArg]
	if !exists {
		return types.IntellijDownloadURL{}, fmt.Errorf("ide download url not found")
	}

	downloadURLStr, ok := downloadURL.(types.IntellijDownloadURL)
	if !ok {
		return types.IntellijDownloadURL{}, fmt.Errorf("ide download url is not of type IntellijDownloadURL")
	}

	return downloadURLStr, nil
}

func getIDEDirName(args map[gitspaceTypes.IDEArg]interface{}) (string, error) {
	dirName, exists := args[gitspaceTypes.IDEDIRNameArg]
	if !exists {
		return "", fmt.Errorf("ide dirname not found")
	}

	dirNameStr, ok := dirName.(string)
	if !ok {
		return "", fmt.Errorf("ide dirname is not of type string")
	}

	return dirNameStr, nil
}

func getRepoName(
	args map[gitspaceTypes.IDEArg]interface{},
) (string, error) {
	repoName, exists := args[gitspaceTypes.IDERepoNameArg]
	if !exists {
		return "", nil // No repo name found, nothing to do
	}

	repoNameStr, ok := repoName.(string)
	if !ok {
		return "", fmt.Errorf("repo name is not of type string")
	}

	return repoNameStr, nil
}
