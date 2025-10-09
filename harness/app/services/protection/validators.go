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

func validateIdentifierSlice(identifiers []string) error {
	if len(identifiers) > maxElements {
		return errors.New("too many Identifiers provided")
	}

	for _, identifier := range identifiers {
		if identifier == "" {
			return errors.New("identifier mustn't be an empty string")
		}
	}

	return nil
}
