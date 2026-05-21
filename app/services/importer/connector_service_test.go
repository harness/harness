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

package importer

import (
	"context"
	"testing"

	"github.com/harness/gitness/errors"
)

func TestConnectorServiceNoop_FetchProviderRepoInfo(t *testing.T) {
	info, err := connectorServiceNoop{}.FetchProviderRepoInfo(context.Background(), ConnectorDef{})
	if err == nil {
		t.Fatal("expected error from noop, got nil")
	}
	if !errors.IsInvalidArgument(err) {
		t.Errorf("expected InvalidArgument error, got status %q: %v", errors.AsStatus(err), err)
	}
	if info != (ProviderRepoInfo{}) {
		t.Errorf("expected zero ProviderRepoInfo, got %+v", info)
	}
}
