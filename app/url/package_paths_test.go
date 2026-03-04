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

package url

import (
	"context"
	"testing"
)

func TestPackagePathFor_Relative(t *testing.T) {
	tests := []struct {
		name    string
		spec    PackagePathSpec
		want    string
		wantErr bool
	}{
		{
			name: "python file path",
			spec: PythonFilePathSpec{
				RegIdentifier: "my-registry",
				Image:         "requests",
				Version:       "2.28.0",
				Filename:      "requests-2.28.0.tar.gz",
			},
			want: "../../_/my-registry/requests/2.28.0/requests-2.28.0.tar.gz",
		},
		{
			name: "python file path with wheel",
			spec: PythonFilePathSpec{
				RegIdentifier: "pypi-proxy",
				Image:         "numpy",
				Version:       "1.24.3",
				Filename:      "numpy-1.24.3-cp311-cp311-manylinux_2_17_x86_64.whl",
			},
			want: "../../_/pypi-proxy/numpy/1.24.3/numpy-1.24.3-cp311-cp311-manylinux_2_17_x86_64.whl",
		},
	}

	p, err := NewProvider(
		"http://localhost", "http://localhost",
		"http://localhost", "http://localhost",
		"", "", false,
		"http://localhost", "https://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.PackagePathFor(ctx, tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackagePathFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PackagePathFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
