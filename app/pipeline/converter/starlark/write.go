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

package starlark

import (
	"encoding/json"
	"fmt"
	"io"

	"go.starlark.net/starlark"
)

type writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
}

//nolint:gocognit,cyclop
func write(out writer, v starlark.Value) error {
	if marshaler, ok := v.(json.Marshaler); ok {
		jsonData, err := marshaler.MarshalJSON()
		if err != nil {
			return err
		}
		if _, err = out.Write(jsonData); err != nil {
			return err
		}
		return nil
	}

	switch v := v.(type) {
	case starlark.NoneType:
		if _, err := out.WriteString("null"); err != nil {
			return err
		}
	case starlark.Bool:
		fmt.Fprintf(out, "%t", v)
	case starlark.Int:
		if _, err := out.WriteString(v.String()); err != nil {
			return err
		}
	case starlark.Float:
		fmt.Fprintf(out, "%g", v)
	case starlark.String:
		s := string(v)
		if isQuoteSafe(s) {
			fmt.Fprintf(out, "%q", s)
		} else {
			data, err := json.Marshal(s)
			if err != nil {
				return err
			}
			if _, err = out.Write(data); err != nil {
				return err
			}
		}
	case starlark.Indexable:
		if err := out.WriteByte('['); err != nil {
			return err
		}
		for i, n := 0, starlark.Len(v); i < n; i++ {
			if i > 0 {
				if _, err := out.WriteString(", "); err != nil {
					return err
				}
			}
			if err := write(out, v.Index(i)); err != nil {
				return err
			}
		}
		if err := out.WriteByte(']'); err != nil {
			return err
		}
	case *starlark.Dict:
		if err := out.WriteByte('{'); err != nil {
			return err
		}
		for i, itemPair := range v.Items() {
			key := itemPair[0]
			value := itemPair[1]
			if i > 0 {
				if _, err := out.WriteString(", "); err != nil {
					return err
				}
			}
			if err := write(out, key); err != nil {
				return err
			}
			if _, err := out.WriteString(": "); err != nil {
				return err
			}
			if err := write(out, value); err != nil {
				return err
			}
		}
		if err := out.WriteByte('}'); err != nil {
			return err
		}
	default:
		return fmt.Errorf("value %s (type `%s') can't be converted to JSON", v.String(), v.Type())
	}
	return nil
}

func isQuoteSafe(s string) bool {
	for _, r := range s {
		if r < 0x20 || r >= 0x10000 {
			return false
		}
	}
	return true
}
