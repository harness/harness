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

import (
	"bytes"
	"encoding/json"
)

// ToJSON is utility function that converts types to a JSON message.
// It's used to sanitize protection definition data.
func ToJSON(v any) (json.RawMessage, error) {
	buffer := bytes.NewBuffer(nil)

	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	data := buffer.Bytes()
	data = bytes.TrimSpace(data)

	return data, nil
}
