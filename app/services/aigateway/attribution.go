// Copyright 2026 Harness, Inc.
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

package aigateway

import "context"

type attributionKey struct{}

type attribution struct {
	spaceID *int64
	repoID  *int64
}

// WithAttribution returns a context that makes the gateway attribute the usage records of
// the requests made with it to the space and repo. A zero id leaves that field unset.
func WithAttribution(ctx context.Context, spaceID, repoID int64) context.Context {
	a := attribution{}
	if spaceID != 0 {
		a.spaceID = &spaceID
	}
	if repoID != 0 {
		a.repoID = &repoID
	}
	return context.WithValue(ctx, attributionKey{}, a)
}

// AttributionFrom returns the space and repo set with WithAttribution; nil when unset.
func AttributionFrom(ctx context.Context) (spaceID, repoID *int64) {
	a, _ := ctx.Value(attributionKey{}).(attribution)
	return a.spaceID, a.repoID
}
