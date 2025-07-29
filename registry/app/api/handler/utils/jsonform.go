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

package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
)

// fillFromForm uses reflection to fill fields of 'data' from r.FormValue.
// It looks for a 'json' struct tag and uses that as the key in FormValue.
func FillFromForm(r *http.Request, data interface{}) error {
	// Make sure form data is parsed
	if err := r.ParseForm(); err != nil {
		return err
	}
	if err := r.ParseMultipartForm(32 << 22); err != nil {
		return err
	}

	v := reflect.ValueOf(data).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			// Skip fields with no json tag
			continue
		}

		// The tag might be `json:"author,omitempty"`, so split on comma to isolate the key
		tagParts := strings.Split(jsonTag, ",")
		key := tagParts[0]

		// Single-value retrieval:
		formVal := r.FormValue(key)

		// Now decide how to set based on the field type
		fieldVal := v.Field(i)
		switch fieldVal.Kind() { // nolint:exhaustive
		case reflect.String:
			// Just set the string
			fieldVal.SetString(formVal)

		case reflect.Slice:
			// Check if it's a slice of strings
			if fieldVal.Type().Elem().Kind() == reflect.String {
				// For slices, let's fetch all form values under that key
				// e.g. name=foo&name=bar => r.Form["name"] = []string{"foo","bar"}
				values := r.Form[key]
				fieldVal.Set(reflect.ValueOf(values))
			}

		case reflect.Map:
			// Check if it's a map[string]string
			if fieldVal.Type().Key().Kind() == reflect.String && // nolint:nestif
				fieldVal.Type().Elem().Kind() == reflect.String {
				// We'll assume the form value is a JSON string. For example:
				// extra={"foo":"bar","something":"else"}
				if formVal == "" {
					// If nothing is provided, just set an empty map
					fieldVal.Set(reflect.ValueOf(map[string]string{}))
				} else {
					m := make(map[string]string)
					if err := json.Unmarshal([]byte(formVal), &m); err != nil {
						splitVal := strings.Split(formVal, ", ")
						if len(splitVal) > 1 {
							m[splitVal[0]] = splitVal[1]
						} else {
							return fmt.Errorf("cannot unmarshal map from key %q: %w", key, err)
						}
					}
					fieldVal.Set(reflect.ValueOf(m))
				}
			}
		default:
			log.Warn().Ctx(r.Context()).Msgf("Unsupported field type %v", fieldVal.Kind())
		}
	}
	return nil
}
