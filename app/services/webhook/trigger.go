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

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/harness/gitness/version"

	"github.com/rs/zerolog/log"
)

const (
	// webhookTimeLimit defines the time limit of a single webhook execution.
	// This is similar to other SCM providers.
	webhookTimeLimit = 10 * time.Second

	// responseHeadersBytesLimit defines the maximum number of bytes processed from the webhook response headers.
	responseHeadersBytesLimit = 1024

	// responseBodyBytesLimit defines the maximum number of bytes processed from the webhook response body.
	responseBodyBytesLimit = 1024
)

var (
	// ErrWebhookNotRetriggerable is returned in case the webhook can't be retriggered due to an incomplete execution.
	// This should only occur if we failed to generate the request body (most likely out of memory).
	ErrWebhookNotRetriggerable = errors.New("webhook execution is incomplete and can't be retriggered")
)

type TriggerResult struct {
	TriggerID   string
	TriggerType enum.WebhookTrigger
	Webhook     *types.Webhook
	Execution   *types.WebhookExecution
	Err         error
}

func (r *TriggerResult) Skipped() bool {
	return r.Execution == nil
}

func (s *Service) triggerWebhooksFor(
	ctx context.Context,
	parents []types.WebhookParentInfo,
	triggerID string,
	triggerType enum.WebhookTrigger,
	body any,
) ([]TriggerResult, error) {
	webhooks, err := s.webhookStore.List(ctx, parents, &types.WebhookFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks for: %w", err)
	}

	return s.triggerWebhooks(ctx, webhooks, triggerID, triggerType, body)
}

//nolint:gocognit // refactor if needed
func (s *Service) triggerWebhooks(ctx context.Context, webhooks []*types.Webhook,
	triggerID string, triggerType enum.WebhookTrigger, body any) ([]TriggerResult, error) {
	// return immediately if webhooks are empty
	if len(webhooks) == 0 {
		return []TriggerResult{}, nil
	}

	// get all previous execution for the same trigger
	executions, err := s.webhookExecutionStore.ListForTrigger(ctx, triggerID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to get executions for trigger '%s'", triggerID)
	}

	// precalculate whether a webhook should be executed
	skipExecution := make(map[int64]bool)
	for _, execution := range executions {
		// skip execution in case of success or unrecoverable error
		if execution.Result == enum.WebhookExecutionResultSuccess ||
			execution.Result == enum.WebhookExecutionResultFatalError {
			skipExecution[execution.WebhookID] = true
		}
	}

	results := make([]TriggerResult, len(webhooks))
	for i, webhook := range webhooks {
		results[i] = TriggerResult{
			TriggerID:   triggerID,
			TriggerType: triggerType,
			Webhook:     webhook,
		}

		// check if webhook is disabled
		if !webhook.Enabled {
			continue
		}

		// check if webhook already got executed (success or fatal error)
		if skipExecution[webhook.ID] {
			continue
		}

		// check if webhook is registered for trigger (empty list => all triggers are registered)
		triggerRegistered := len(webhook.Triggers) == 0
		for _, trigger := range webhook.Triggers {
			if trigger == triggerType {
				triggerRegistered = true
				break
			}
		}
		if !triggerRegistered {
			continue
		}

		// execute trigger and store output in result
		results[i].Execution, results[i].Err = s.executeWebhook(ctx, webhook, triggerID, triggerType, body, nil)
	}

	return results, nil
}

func (s *Service) RetriggerWebhookExecution(ctx context.Context, webhookExecutionID int64) (*TriggerResult, error) {
	// find execution
	webhookExecution, err := s.webhookExecutionStore.Find(ctx, webhookExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find webhook execution with id %d: %w", webhookExecutionID, err)
	}

	// ensure webhook can be retriggered
	if !webhookExecution.Retriggerable {
		return nil, ErrWebhookNotRetriggerable
	}

	// find webhook
	webhook, err := s.webhookStore.Find(ctx, webhookExecution.WebhookID)
	if err != nil {
		return nil, fmt.Errorf("failed to find webhook with id %d: %w", webhookExecution.WebhookID, err)
	}

	// reuse same trigger id as original execution
	triggerID := webhookExecution.TriggerID
	triggerType := webhookExecution.TriggerType

	// pass body explicitly
	body := &bytes.Buffer{}
	// NOTE: bBuff.Write(v) will always return (len(v), nil) - no need to error handle
	body.WriteString(webhookExecution.Request.Body)

	newExecution, err := s.executeWebhook(ctx, webhook, triggerID, triggerType, body, &webhookExecution.ID)
	return &TriggerResult{
		TriggerID:   triggerID,
		TriggerType: triggerType,
		Webhook:     webhook,
		Execution:   newExecution,
		Err:         err,
	}, nil
}

//nolint:gocognit // refactor into smaller chunks if necessary.
func (s *Service) executeWebhook(ctx context.Context, webhook *types.Webhook, triggerID string,
	triggerType enum.WebhookTrigger, body any, rerunOfID *int64) (*types.WebhookExecution, error) {
	// build execution entry on the fly (save no matter what)
	execution := types.WebhookExecution{
		RetriggerOf: rerunOfID,
		WebhookID:   webhook.ID,
		TriggerID:   triggerID,
		TriggerType: triggerType,
		// for unexpected errors we don't retry - protect the system. User can retrigger manually (if body was set)
		Result: enum.WebhookExecutionResultFatalError,
		Error:  "An unknown error occurred",
	}
	defer func(oCtx context.Context, start time.Time) {
		// set total execution time
		execution.Duration = int64(time.Since(start))
		execution.Created = time.Now().UnixMilli()

		// TODO: what if saving execution failed? For now we will rerun it in case of error or not show it in history
		err := s.webhookExecutionStore.Create(oCtx, &execution)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf(
				"failed to store webhook execution that ended with Result: %s, Response.Status: '%s', Error: '%s'",
				execution.Result, execution.Response.Status, execution.Error)
		}

		// update latest execution result of webhook IFF it's different from before (best effort)
		if webhook.LatestExecutionResult == nil || *webhook.LatestExecutionResult != execution.Result {
			_, err = s.webhookStore.UpdateOptLock(oCtx, webhook, func(hook *types.Webhook) error {
				hook.LatestExecutionResult = &execution.Result
				return nil
			})
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf(
					"failed to update latest execution result to %s for webhook %d",
					execution.Result, webhook.ID)
			}
		}
	}(ctx, time.Now())

	// derive context with time limit
	ctx, cancel := context.WithTimeout(ctx, webhookTimeLimit)
	defer cancel()

	// create request from webhook and body
	req, err := s.prepareHTTPRequest(ctx, &execution, triggerType, webhook, body)
	if err != nil {
		return &execution, err
	}

	// Execute HTTP Request (insecure if requested)
	var resp *http.Response
	switch {
	case webhook.Internal && webhook.Insecure:
		resp, err = s.insecureHTTPClientInternal.Do(req)
	case webhook.Internal:
		resp, err = s.secureHTTPClientInternal.Do(req)
	case webhook.Insecure:
		resp, err = s.insecureHTTPClient.Do(req)
	default:
		resp, err = s.secureHTTPClient.Do(req)
	}

	// always close the body!
	if resp != nil && resp.Body != nil {
		defer func() {
			err = resp.Body.Close()
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf("failed to close body after webhook execution %d", execution.ID)
			}
		}()
	}

	// handle certain errors explicitly to give more to-the-point error messages
	var dnsError *net.DNSError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		// we assume timeout without any response is not worth retrying - protect the system
		tErr := fmt.Errorf("request exceeded time limit of %s", webhookTimeLimit)
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultFatalError
		return &execution, tErr

	case errors.As(err, &dnsError) && dnsError.IsNotFound:
		// this error is assumed unrecoverable - mark status accordingly and fail execution
		execution.Error = fmt.Sprintf("host '%s' was not found", dnsError.Name)
		execution.Result = enum.WebhookExecutionResultFatalError
		return &execution, fmt.Errorf("failed to resolve host name '%s': %w", dnsError.Name, err)

	case err != nil:
		// for all other errors we don't retry - protect the system. User can retrigger manually (if body was set)
		tErr := fmt.Errorf("an error occurred while sending the request: %w", err)
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultFatalError
		return &execution, tErr
	}

	// handle response
	err = handleWebhookResponse(&execution, resp)

	return &execution, err
}

// prepareHTTPRequest prepares a new http.Request object for the webhook using the provided body as request body.
// All execution.Request.XXX values are set accordingly.
// NOTE: if the body is an io.Reader, the value is used as response body as is, otherwise it'll be JSON serialized.
func (s *Service) prepareHTTPRequest(ctx context.Context, execution *types.WebhookExecution,
	triggerType enum.WebhookTrigger, webhook *types.Webhook, body any) (*http.Request, error) {
	url, err := s.webhookURLProvider.GetWebhookURL(ctx, webhook)
	if err != nil {
		return nil, fmt.Errorf("webhook url is not resolvable: %w", err)
	}
	execution.Request.URL = url

	// Serialize body before anything else.
	// This allows the user to retrigger the execution even in case of bad URL.
	bBuff := &bytes.Buffer{}
	switch v := body.(type) {
	case io.Reader:
		// if it's already an io.Reader - use value as is and don't serialize (allows to provide custom body)
		// NOTE: reader can be read only once - read and store it in buffer to allow storing it in execution object
		// and generate hmac.
		bBytes, err := io.ReadAll(v)
		if err != nil {
			// ASSUMPTION: there was an issue with the static user input, not retriable
			tErr := fmt.Errorf("failed to generate request body: %w", err)
			execution.Error = tErr.Error()
			execution.Result = enum.WebhookExecutionResultFatalError
			return nil, tErr
		}

		// NOTE: bBuff.Write(v) will always return (len(v), nil) - no need to error handle
		bBuff.Write(bBytes)

	default:
		// all other types we json serialize
		err := json.NewEncoder(bBuff).Encode(body)
		if err != nil {
			// this is an internal issue, nothing the user can do - don't expose error details
			execution.Error = "an error occurred preparing the request body"
			execution.Result = enum.WebhookExecutionResultFatalError
			return nil, fmt.Errorf("failed to serialize body to json: %w", err)
		}
	}
	// set executioon body and mark it as retriggerable
	execution.Request.Body = bBuff.String()
	execution.Retriggerable = true

	// create request (url + body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bBuff)
	if err != nil {
		// ASSUMPTION: there was an issue with the static user input, not retriable
		tErr := fmt.Errorf("failed to create request: %w", err)
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultFatalError
		return nil, tErr
	}

	// setup headers
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", s.config.UserAgentIdentity, version.Version))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(s.toXHeader("Trigger"), string(triggerType))
	req.Header.Add(s.toXHeader("Webhook-Parent-Type"), string(webhook.ParentType))
	req.Header.Add(s.toXHeader("Webhook-Parent-Id"), fmt.Sprint(webhook.ParentID))
	// TODO [CODE-1363]: remove after identifier migration.
	req.Header.Add(s.toXHeader("Webhook-Uid"), fmt.Sprint(webhook.Identifier))
	req.Header.Add(s.toXHeader("Webhook-Identifier"), fmt.Sprint(webhook.Identifier))

	// add HMAC only if a secret was provided
	if webhook.Secret != "" {
		decryptedSecret, err := s.encrypter.Decrypt([]byte(webhook.Secret))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt webhook secret: %w", err)
		}
		var hmac string
		hmac, err = generateHMACSHA256(bBuff.Bytes(), []byte(decryptedSecret))
		if err != nil {
			return nil, fmt.Errorf("failed to generate SHA256 based HMAC: %w", err)
		}
		req.Header.Add(s.toXHeader("Signature"), hmac)
	}

	hBuffer := &bytes.Buffer{}
	err = req.Header.Write(hBuffer)
	if err != nil {
		tErr := fmt.Errorf("failed to write request headers: %w", err)
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultRetriableError
		return nil, tErr
	}
	execution.Request.Headers = hBuffer.String()

	return req, nil
}

func (s *Service) toXHeader(name string) string {
	return fmt.Sprintf("X-%s-%s", s.config.HeaderIdentity, name)
}

//nolint:funlen // refactor if needed
func handleWebhookResponse(execution *types.WebhookExecution, resp *http.Response) error {
	// store status (handle status later - want to first read body)
	execution.Response.StatusCode = resp.StatusCode
	execution.Response.Status = resp.Status

	// store response headers
	hBuff := &bytes.Buffer{}
	err := resp.Header.Write(hBuff)
	if err != nil {
		tErr := fmt.Errorf("failed to read response headers: %w", err)
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultRetriableError
		return tErr
	}
	// limit the total number of bytes we store in headers
	headerLength := hBuff.Len()
	if headerLength > responseHeadersBytesLimit {
		headerLength = responseHeadersBytesLimit
	}
	execution.Response.Headers = string(hBuff.Bytes()[0:headerLength])

	// handle body (if exists)
	if resp.Body != nil {
		// read and store response body
		var bodyRaw []byte
		bodyRaw, err = io.ReadAll(io.LimitReader(resp.Body, responseBodyBytesLimit))
		if err != nil {
			tErr := fmt.Errorf("an error occurred while reading the response body: %w", err)
			execution.Error = tErr.Error()
			execution.Result = enum.WebhookExecutionResultRetriableError
			return tErr
		}
		execution.Response.Body = string(bodyRaw)
	}

	// Analyze status code
	// IMPORTANT: cases are EVALUATED IN ORDER
	switch code := resp.StatusCode; {
	case code < 200:
		// 1XX - server is continuing the processing (call was successful, but not completed yet)
		execution.Error = "1xx response codes are not supported"
		execution.Result = enum.WebhookExecutionResultFatalError
		return fmt.Errorf("received response with unsupported status code %d", code)

	case code < 300:
		// 2XX - call was successful
		execution.Error = ""
		execution.Result = enum.WebhookExecutionResultSuccess
		return nil

	case code < 400:
		// 3XX - Redirection (further action is required by the client)
		// NOTE: technically we could follow the redirect, but not supported as of now
		execution.Error = "3xx response codes are not supported"
		execution.Result = enum.WebhookExecutionResultFatalError
		return fmt.Errorf("received response with unsupported status code %d", code)

	case code == 408:
		// 408 - Request Timeout
		tErr := errors.New("request timed out")
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultRetriableError
		return tErr

	case code == 429:
		// 429 - Too Many Requests
		tErr := errors.New("request got throttled")
		execution.Error = tErr.Error()
		execution.Result = enum.WebhookExecutionResultRetriableError
		return tErr

	case code < 500:
		// 4xx - Issue with request (bad request, url too large, ...)
		execution.Error = "4xx response codes are not supported (apart from 408 and 429)"
		execution.Result = enum.WebhookExecutionResultFatalError
		return fmt.Errorf("received response with unrecoverable status code %d", code)

	case code == 501:
		// 501 - Not Implemented
		execution.Error = "remote server does not implement requested action"
		execution.Result = enum.WebhookExecutionResultFatalError
		return fmt.Errorf("received response with unrecoverable status code %d", code)

	case code < 600:
		// 5xx - Server Errors
		execution.Error = "remote server encountered an error"
		execution.Result = enum.WebhookExecutionResultRetriableError
		return fmt.Errorf("remote server encountered an error: %d", code)

	default:
		// >= 600 - No commonly used response status code
		execution.Error = "response code not supported"
		execution.Result = enum.WebhookExecutionResultFatalError
		return fmt.Errorf("received response with unsupported status code %d", code)
	}
}

// generateHMACSHA256 generates a new HMAC using SHA256 as hash function.
func generateHMACSHA256(data []byte, key []byte) (string, error) {
	h := hmac.New(sha256.New, key)

	// write all data into hash
	_, err := h.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write data into hash: %w", err)
	}

	// sum hash to final value
	macBytes := h.Sum(nil)

	// encode MAC as hexadecimal
	return hex.EncodeToString(macBytes), nil
}
