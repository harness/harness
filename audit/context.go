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

package audit

import "context"

type key int

const (
	realIPKey key = iota
	requestID
	requestMethod
)

// GetRealIP returns IP address from context.
func GetRealIP(ctx context.Context) string {
	ip, ok := ctx.Value(realIPKey).(string)
	if !ok {
		return ""
	}

	return ip
}

// GetRequestID returns requestID from context.
func GetRequestID(ctx context.Context) string {
	id, ok := ctx.Value(requestID).(string)
	if !ok {
		return ""
	}

	return id
}

// GetRequestMethod returns http method from context.
func GetRequestMethod(ctx context.Context) string {
	method, ok := ctx.Value(requestMethod).(string)
	if !ok {
		return ""
	}

	return method
}
