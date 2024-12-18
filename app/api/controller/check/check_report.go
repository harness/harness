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

package check

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ReportInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	CheckUID   string             `json:"check_uid" deprecated:"true"`
	Identifier string             `json:"identifier"`
	Status     enum.CheckStatus   `json:"status"`
	Summary    string             `json:"summary"`
	Link       string             `json:"link"`
	Payload    types.CheckPayload `json:"payload"`

	Started int64 `json:"started,omitempty"`
	Ended   int64 `json:"ended,omitempty"`
}

// TODO: Can we drop the '$' - depends on whether harness allows it.
var regexpCheckIdentifier = "^[0-9a-zA-Z-_.$]{1,127}$"
var matcherCheckIdentifier = regexp.MustCompile(regexpCheckIdentifier)

// Sanitize validates and sanitizes the ReportInput data.
func (in *ReportInput) Sanitize(
	sanitizers map[enum.CheckPayloadKind]func(in *ReportInput, s *auth.Session) error, session *auth.Session,
) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.CheckUID
	}

	if in.Identifier == "" {
		return usererror.BadRequest("Identifier is missing")
	}

	if !matcherCheckIdentifier.MatchString(in.Identifier) {
		return usererror.BadRequestf("Identifier must match the regular expression: %s", regexpCheckIdentifier)
	}

	_, ok := in.Status.Sanitize()
	if !ok {
		return usererror.BadRequest("Invalid value provided for status check status")
	}

	validatorFn, ok := sanitizers[in.Payload.Kind]
	if !ok {
		return usererror.BadRequest("Invalid value provided for the payload kind")
	}

	// Validate and sanitize the input data based on version; Require a link... and similar operations.
	if err := validatorFn(in, session); err != nil {
		return fmt.Errorf("payload validation failed: %w", err)
	}

	if in.Ended != 0 && in.Ended < in.Started {
		return usererror.BadRequest("started time reported after ended time")
	}

	return nil
}

func SanitizeJSONPayload(source json.RawMessage, data any) (json.RawMessage, error) {
	if len(source) == 0 {
		return json.Marshal(data) // marshal the empty object
	}

	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&data); err != nil {
		return nil, usererror.BadRequestf("Payload data doesn't match the required format: %s", err.Error())
	}

	buffer := bytes.NewBuffer(nil)
	buffer.Grow(512)

	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to sanitize json payload: %w", err)
	}

	result := buffer.Bytes()

	if result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Report modifies an existing or creates a new (if none yet exists) status check report for a specific commit.
func (c *Controller) Report(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	commitSHA string,
	in *ReportInput,
	metadata map[string]string,
) (*types.Check, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReportCommitCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if errValidate := in.Sanitize(c.sanitizers, session); errValidate != nil {
		return nil, errValidate
	}

	if !git.ValidateCommitSHA(commitSHA) {
		return nil, usererror.BadRequest("invalid commit SHA provided")
	}

	_, err = c.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Revision:   commitSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to commit sha=%s: %w", commitSHA, err)
	}

	now := time.Now().UnixMilli()

	metadataJSON, _ := json.Marshal(metadata)

	existingCheck, err := c.checkStore.FindByIdentifier(ctx, repo.ID, commitSHA, in.Identifier)

	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to find existing check for Identifier %q: %w", in.Identifier, err)
	}

	started := getStartTime(in, existingCheck, now)
	ended := getEndTime(in, now)

	statusCheckReport := &types.Check{
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		RepoID:     repo.ID,
		CommitSHA:  commitSHA,
		Identifier: in.Identifier,
		Status:     in.Status,
		Summary:    in.Summary,
		Link:       in.Link,
		Payload:    in.Payload,
		Metadata:   metadataJSON,
		ReportedBy: session.Principal.ToPrincipalInfo(),
		Started:    started,
		Ended:      ended,
	}

	err = c.checkStore.Upsert(ctx, statusCheckReport)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert status check result for repo=%s: %w", repo.Identifier, err)
	}

	c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeStatusCheckReportUpdated, statusCheckReport)

	return statusCheckReport, nil
}

func getStartTime(in *ReportInput, check types.Check, now int64) int64 {
	// start value came in api
	if in.Started != 0 {
		return in.Started
	}
	// in.started has no value we smartly put value for started

	// in case of pending we assume check has not started running
	if in.Status == enum.CheckStatusPending {
		return 0
	}

	// new check
	if check.Started == 0 {
		return now
	}

	// The incoming check status can now be running or terminal.

	// in case we already have running status we don't update time else we return current time as check has started
	// running.
	if check.Status == enum.CheckStatusRunning {
		return check.Started
	}

	// Note: In case of reporting terminal statuses again and again we have assumed its
	// a report of new status check everytime.

	// In case someone reports any status before marking running return current time.
	// This can happen if someone only reports terminal status or marks running status again after terminal.
	return now
}

func getEndTime(in *ReportInput, now int64) int64 {
	// end value came in api
	if in.Ended != 0 {
		return in.Ended
	}

	// if we get terminal status i.e. error, failure or success we return current time.
	if in.Status.IsCompleted() {
		return now
	}

	// in case of other status we return value as 0, which means we have not yet completed the check.
	return 0
}
