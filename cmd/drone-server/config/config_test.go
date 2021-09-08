// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import "testing"

func Test_cleanHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{
			name:     "no prefix",
			hostname: "drone.io",
			want:     "drone.io",
		},
		{
			name:     "http prefix",
			hostname: "http://drone.io",
			want:     "drone.io",
		},
		{
			name:     "https prefix",
			hostname: "https://drone.io",
			want:     "drone.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanHostname(tt.hostname); got != tt.want {
				t.Errorf("cleanHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}
