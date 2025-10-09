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
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

// Helper functions for creating pointers.
func StringPtr(s string) *string                                          { return &s }
func Int64Ptr(i int64) *int64                                             { return &i }
func IntPtr(i int) *int                                                   { return &i }
func BoolPtr(b bool) *bool                                                { return &b }
func Int32Ptr(i int32) *int32                                             { return &i }
func WebhookExecResultPtr(r api.WebhookExecResult) *api.WebhookExecResult { return &r }
func WebhookTriggerPtr(t api.Trigger) *api.Trigger                        { return &t }
func PageSizePtr(i int32) *api.PageSize                                   { size := api.PageSize(i); return &size }
func PageNumberPtr(i int32) *api.PageNumber                               { num := api.PageNumber(i); return &num }
