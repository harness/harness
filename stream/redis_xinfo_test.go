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

package stream

import (
	"reflect"
	"testing"
)

func TestParseXInfoConsumers(t *testing.T) {
	tests := []struct {
		name    string
		reply   interface{}
		want    []xInfoConsumer
		wantErr bool
	}{
		{
			name:  "empty group",
			reply: []interface{}{},
			want:  []xInfoConsumer{},
		},
		{
			// reply shape of redis < 7.2
			name: "without inactive field",
			reply: []interface{}{
				[]interface{}{"name", "c1", "pending", int64(1), "idle", int64(101)},
			},
			want: []xInfoConsumer{{Name: "c1", Pending: 1, Idle: 101}},
		},
		{
			// reply shape of redis >= 7.2 - the inactive field is ignored
			name: "with inactive field",
			reply: []interface{}{
				[]interface{}{"name", "c1", "pending", int64(1), "idle", int64(101), "inactive", int64(102)},
			},
			want: []xInfoConsumer{{Name: "c1", Pending: 1, Idle: 101}},
		},
		{
			// any field added by future redis versions has to be ignored, too
			name: "with unknown fields",
			reply: []interface{}{
				[]interface{}{
					"name", "c1", "unknown", "whatever", "pending", int64(1), "idle", int64(101), "inactive", int64(102),
				},
			},
			want: []xInfoConsumer{{Name: "c1", Pending: 1, Idle: 101}},
		},
		{
			name: "multiple consumers",
			reply: []interface{}{
				[]interface{}{"name", "c1", "pending", int64(0), "idle", int64(1), "inactive", int64(1)},
				[]interface{}{"name", "c2", "pending", int64(5), "idle", int64(60000), "inactive", int64(60000)},
			},
			want: []xInfoConsumer{
				{Name: "c1", Pending: 0, Idle: 1},
				{Name: "c2", Pending: 5, Idle: 60000},
			},
		},
		{
			name: "numeric values as strings",
			reply: []interface{}{
				[]interface{}{"name", "c1", "pending", "1", "idle", "101"},
			},
			want: []xInfoConsumer{{Name: "c1", Pending: 1, Idle: 101}},
		},
		{
			name:    "reply is not an array",
			reply:   "nope",
			wantErr: true,
		},
		{
			name:    "entry is not an array",
			reply:   []interface{}{"nope"},
			wantErr: true,
		},
		{
			name:    "entry has odd number of elements",
			reply:   []interface{}{[]interface{}{"name", "c1", "pending"}},
			wantErr: true,
		},
		{
			name:    "entry without name",
			reply:   []interface{}{[]interface{}{"pending", int64(1), "idle", int64(101)}},
			wantErr: true,
		},
		{
			name:    "entry with non-numeric pending",
			reply:   []interface{}{[]interface{}{"name", "c1", "pending", "abc", "idle", int64(101)}},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseXInfoConsumers(test.reply)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got consumers %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("expected consumers %v, got %v", test.want, got)
			}
		})
	}
}
