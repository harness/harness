// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
