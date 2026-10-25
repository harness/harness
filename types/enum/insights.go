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

package enum

// InsightsIssueStatus defines the lifecycle status of an Insights issue.
type InsightsIssueStatus string

func (InsightsIssueStatus) Enum() []any { return toInterfaceSlice(insightsIssueStatuses) }
func (s InsightsIssueStatus) Sanitize() (InsightsIssueStatus, bool) {
	return Sanitize(s, GetAllInsightsIssueStatuses)
}
func GetAllInsightsIssueStatuses() ([]InsightsIssueStatus, InsightsIssueStatus) {
	return insightsIssueStatuses, InsightsIssueStatusUnresolved
}

// InsightsIssueStatus enumeration.
const (
	InsightsIssueStatusUnresolved InsightsIssueStatus = "unresolved"
	InsightsIssueStatusResolved   InsightsIssueStatus = "resolved"
	InsightsIssueStatusIgnored    InsightsIssueStatus = "ignored"
	InsightsIssueStatusRegressed  InsightsIssueStatus = "regressed"
)

var insightsIssueStatuses = sortEnum([]InsightsIssueStatus{
	InsightsIssueStatusUnresolved,
	InsightsIssueStatusResolved,
	InsightsIssueStatusIgnored,
	InsightsIssueStatusRegressed,
})
