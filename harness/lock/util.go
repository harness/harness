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

package lock

import "strings"

func formatKey(app, ns, key string) string {
	return app + ":" + ns + ":" + key
}

func SplitKey(uniqKey string) (namespace, key string) {
	parts := strings.Split(uniqKey, ":")
	key = uniqKey
	if len(parts) > 2 {
		namespace = parts[1]
		key = parts[2]
	}
	return
}
