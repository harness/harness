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

package protection

import "errors"

const maxElements = 100

func validateIDSlice(ids []int64) error {
	if len(ids) > maxElements {
		return errors.New("too many IDs provided")
	}

	for _, id := range ids {
		if id <= 0 {
			return errors.New("ID must be a positive integer")
		}
	}

	return nil
}

func validateUIDSlice(uids []string) error {
	if len(uids) > maxElements {
		return errors.New("too many UIDs provided")
	}

	for _, uid := range uids {
		if uid == "" {
			return errors.New("UID mustn't be an empty string")
		}
	}

	return nil
}
