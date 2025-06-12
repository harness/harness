//  Copyright 2023 Harness, Inc.
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

package cargo

import "fmt"

func getCrateFileName(imageName string, version string) string {
	return fmt.Sprintf("%s-%s.crate", imageName, version)
}

func getCrateFilePath(imageName string, version string) string {
	return fmt.Sprintf("/crates/%s/%s/%s", imageName, version, getCrateFileName(imageName, version))
}
