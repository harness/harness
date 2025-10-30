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

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamWebhookIdentifier  = "webhook_identifier"
	PathParamWebhookExecutionID = "webhook_execution_id"
)

func GetWebhookIdentifierFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamWebhookIdentifier)
}

func GetWebhookExecutionIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamWebhookExecutionID)
}

// ParseWebhookFilter extracts the Webhook query parameters for listing from the url.
func ParseWebhookFilter(r *http.Request) *types.WebhookFilter {
	return &types.WebhookFilter{
		Query: ParseQuery(r),
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
		Sort:  ParseSortWebhook(r),
		Order: ParseOrder(r),
	}
}

// ParseWebhookExecutionFilter extracts the WebhookExecution query parameters for listing from the url.
func ParseWebhookExecutionFilter(r *http.Request) *types.WebhookExecutionFilter {
	return &types.WebhookExecutionFilter{
		Page: ParsePage(r),
		Size: ParseLimit(r),
	}
}

// ParseSortWebhook extracts the webhook sort parameter from the url.
func ParseSortWebhook(r *http.Request) enum.WebhookAttr {
	return enum.ParseWebhookAttr(
		r.URL.Query().Get(QueryParamSort),
	)
}
