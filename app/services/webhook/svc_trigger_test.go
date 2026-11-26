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

package webhook

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/crypto"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// svcExecStore is an in-memory WebhookExecutorStore.
type svcExecStore struct {
	mu            sync.Mutex
	webhooks      map[int64]*types.WebhookCore
	executions    map[int64]*types.WebhookExecutionCore
	created       []*types.WebhookExecutionCore
	updated       []enum.WebhookExecutionResult
	listHooksErr  error
	listTrigErr   error
	findExecErr   error
	findHookErr   error
	nextExecution int64
}

func newSvcExecStore() *svcExecStore {
	return &svcExecStore{
		webhooks:   map[int64]*types.WebhookCore{},
		executions: map[int64]*types.WebhookExecutionCore{},
	}
}

func (s *svcExecStore) Find(_ context.Context, id int64) (*types.WebhookExecutionCore, error) {
	if s.findExecErr != nil {
		return nil, s.findExecErr
	}
	e, ok := s.executions[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return e, nil
}

func (s *svcExecStore) ListWebhooks(context.Context, []types.WebhookParentInfo) ([]*types.WebhookCore, error) {
	if s.listHooksErr != nil {
		return nil, s.listHooksErr
	}
	res := make([]*types.WebhookCore, 0, len(s.webhooks))
	for id := int64(1); id <= int64(len(s.webhooks)); id++ {
		if w, ok := s.webhooks[id]; ok {
			res = append(res, w)
		}
	}
	return res, nil
}

func (s *svcExecStore) UpdateOptLock(
	_ context.Context, hook *types.WebhookCore, execution *types.WebhookExecutionCore,
) (*types.WebhookCore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updated = append(s.updated, execution.Result)
	r := execution.Result
	hook.LatestExecutionResult = &r
	return hook, nil
}

func (s *svcExecStore) FindWebhook(_ context.Context, id int64) (*types.WebhookCore, error) {
	if s.findHookErr != nil {
		return nil, s.findHookErr
	}
	w, ok := s.webhooks[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return w, nil
}

func (s *svcExecStore) ListForTrigger(context.Context, string) ([]*types.WebhookExecutionCore, error) {
	if s.listTrigErr != nil {
		return nil, s.listTrigErr
	}
	res := make([]*types.WebhookExecutionCore, 0, len(s.executions))
	for _, e := range s.executions {
		res = append(res, e)
	}
	return res, nil
}

func (s *svcExecStore) CreateWebhookExecution(_ context.Context, e *types.WebhookExecutionCore) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextExecution++
	e.ID = 1000 + s.nextExecution
	cp := *e
	s.created = append(s.created, &cp)
	return nil
}

type svcURLProvider struct {
	url string
	err error
}

func (p svcURLProvider) GetWebhookURL(_ context.Context, w *types.WebhookCore) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	if p.url != "" {
		return p.url, nil
	}
	return w.URL, nil
}

type svcEncrypter struct {
	decErr error
}

func (e svcEncrypter) Encrypt(plaintext string) ([]byte, error) { return []byte(plaintext), nil }
func (e svcEncrypter) Decrypt(ciphertext []byte) (string, error) {
	if e.decErr != nil {
		return "", e.decErr
	}
	return string(ciphertext), nil
}

type svcSecretService struct {
	value string
	err   error
}

func (s svcSecretService) DecryptSecret(_ context.Context, spacePath, identifier string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.value + ":" + spacePath + ":" + identifier, nil
}

type svcSpacePathStore struct {
	appstore.SpacePathStore
	err error
}

func (s svcSpacePathStore) FindPrimaryBySpaceID(_ context.Context, id int64) (*types.SpacePath, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &types.SpacePath{Value: "root/space", SpaceID: id, IsPrimary: true}, nil
}

func svcIsDiscard(err error) bool {
	return errors.Is(err, events.NewDiscardEventError(nil))
}

func svcNewExecutor(
	t *testing.T, st WebhookExecutorStore, urlProvider URLProvider,
) *WebhookExecutor {
	t.Helper()
	cfg := Config{
		UserAgentIdentity:   "Svctest",
		HeaderIdentity:      "Svctest",
		AllowLoopback:       true,
		AllowPrivateNetwork: true,
		InternalSecret:      "internalsecret",
	}
	return NewWebhookExecutor(cfg, urlProvider, svcEncrypter{}, svcSpacePathStore{}, svcSecretService{value: "sv"},
		nil, st, RepoTrigger)
}

type svcCaptured struct {
	mu      sync.Mutex
	headers http.Header
	body    string
	count   int
}

func svcServer(t *testing.T, status int, respBody string) (*httptest.Server, *svcCaptured) {
	t.Helper()
	c := &svcCaptured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.headers = r.Header.Clone()
		c.body = string(b)
		c.count++
		c.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

func TestSvcConfigPrepare(t *testing.T) {
	valid := func() *Config {
		return &Config{EventReaderName: "r", UserAgentIdentity: "ua", Concurrency: 1}
	}
	tests := []struct {
		name    string
		cfg     func() *Config
		wantErr bool
	}{
		{"nil config", func() *Config { return nil }, true},
		{"missing reader name", func() *Config { c := valid(); c.EventReaderName = ""; return c }, true},
		{"missing user agent", func() *Config { c := valid(); c.UserAgentIdentity = ""; return c }, true},
		{"zero concurrency", func() *Config { c := valid(); c.Concurrency = 0; return c }, true},
		{"negative retries", func() *Config { c := valid(); c.MaxRetries = -1; return c }, true},
		{"valid", valid, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg().Prepare()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

	c := valid()
	require.NoError(t, c.Prepare())
	assert.Equal(t, "ua", c.HeaderIdentity, "header identity is backfilled from the user agent identity")
}

func TestSvcHandleWebhookResponse(t *testing.T) {
	tests := []struct {
		code       int
		wantResult enum.WebhookExecutionResult
		wantErr    bool
	}{
		{100, enum.WebhookExecutionResultFatalError, true},
		{200, enum.WebhookExecutionResultSuccess, false},
		{204, enum.WebhookExecutionResultSuccess, false},
		{302, enum.WebhookExecutionResultFatalError, true},
		{400, enum.WebhookExecutionResultFatalError, true},
		{404, enum.WebhookExecutionResultFatalError, true},
		{408, enum.WebhookExecutionResultRetriableError, true},
		{429, enum.WebhookExecutionResultRetriableError, true},
		{500, enum.WebhookExecutionResultRetriableError, true},
		{501, enum.WebhookExecutionResultFatalError, true},
		{503, enum.WebhookExecutionResultRetriableError, true},
		{600, enum.WebhookExecutionResultFatalError, true},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			exec := &types.WebhookExecutionCore{}
			resp := &http.Response{
				StatusCode: tt.code,
				Status:     http.StatusText(tt.code),
				Header:     http.Header{"X-Test": []string{"v"}},
				Body:       io.NopCloser(strings.NewReader("response")),
			}
			err := handleWebhookResponse(exec, resp)
			assert.Equal(t, tt.wantResult, exec.Result)
			assert.Equal(t, tt.code, exec.Response.StatusCode)
			assert.Equal(t, "response", exec.Response.Body)
			assert.Contains(t, exec.Response.Headers, "X-Test")
			if tt.wantErr {
				require.Error(t, err)
				assert.NotEmpty(t, exec.Error)
			} else {
				require.NoError(t, err)
				assert.Empty(t, exec.Error)
			}
		})
	}
}

func TestSvcHandleWebhookResponse_LimitsAndErrors(t *testing.T) {
	t.Run("body is truncated", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("a", responseBodyBytesLimit*3))),
		}
		require.NoError(t, handleWebhookResponse(exec, resp))
		assert.Len(t, exec.Response.Body, responseBodyBytesLimit)
	})
	t.Run("headers are truncated", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Big": []string{strings.Repeat("a", responseHeadersBytesLimit*2)}},
		}
		require.NoError(t, handleWebhookResponse(exec, resp))
		assert.Len(t, exec.Response.Headers, responseHeadersBytesLimit)
	})
	t.Run("body read error is retriable", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(svcErrReader{}),
		}
		err := handleWebhookResponse(exec, resp)
		require.Error(t, err)
		assert.Equal(t, enum.WebhookExecutionResultRetriableError, exec.Result)
	})
}

type svcErrReader struct{}

func (svcErrReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestSvcTriggerWebhooks_Empty(t *testing.T) {
	ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
	res, err := ex.triggerWebhooks(context.Background(), nil, "t1", enum.WebhookTriggerBranchCreated, "body")
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestSvcTriggerWebhooks_SkipAndExecute(t *testing.T) {
	srv, captured := svcServer(t, http.StatusOK, "ok")
	st := newSvcExecStore()
	st.executions[1] = &types.WebhookExecutionCore{
		ID: 1, WebhookID: 3, Result: enum.WebhookExecutionResultSuccess,
	}
	st.executions[2] = &types.WebhookExecutionCore{
		ID: 2, WebhookID: 4, Result: enum.WebhookExecutionResultRetriableError,
	}
	webhooks := []*types.WebhookCore{
		{ID: 1, Enabled: false, URL: srv.URL},
		{ID: 2, Enabled: true, URL: srv.URL, Triggers: []enum.WebhookTrigger{enum.WebhookTriggerTagCreated}},
		{ID: 3, Enabled: true, URL: srv.URL},
		{ID: 4, Enabled: true, URL: srv.URL, Triggers: []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated}},
		{ID: 5, Enabled: true, URL: srv.URL, ParentType: enum.WebhookParentRepo},
	}
	ex := svcNewExecutor(t, st, svcURLProvider{})

	res, err := ex.triggerWebhooks(
		context.Background(), webhooks, "trig", enum.WebhookTriggerBranchCreated, map[string]string{"a": "b"})
	require.NoError(t, err)
	require.Len(t, res, 5)

	assert.True(t, res[0].Skipped(), "disabled webhook")
	assert.True(t, res[1].Skipped(), "trigger not registered")
	assert.True(t, res[2].Skipped(), "previous execution succeeded")
	require.False(t, res[3].Skipped(), "previous retriable error is re-executed")
	require.False(t, res[4].Skipped())
	require.NoError(t, res[4].Err)
	assert.Equal(t, enum.WebhookExecutionResultSuccess, res[4].Execution.Result)

	assert.Equal(t, 2, captured.count)
	assert.JSONEq(t, `{"a":"b"}`, captured.body)
	assert.Equal(t, "application/json", captured.headers.Get("Content-Type"))
	assert.Equal(t, "branch_created", captured.headers.Get("X-Svctest-Trigger"))
	assert.Equal(t, "repo", captured.headers.Get("X-Svctest-Webhook-Parent-Type"))

	require.Len(t, st.created, 2)
	assert.Len(t, st.updated, 2, "latest execution result is persisted on the webhook")
}

func TestSvcTriggerWebhooks_ListForTriggerErrors(t *testing.T) {
	srv, _ := svcServer(t, http.StatusOK, "")
	hooks := []*types.WebhookCore{{ID: 1, Enabled: true, URL: srv.URL}}

	t.Run("not found is ignored", func(t *testing.T) {
		st := newSvcExecStore()
		st.listTrigErr = store.ErrResourceNotFound
		res, err := svcNewExecutor(t, st, svcURLProvider{}).
			triggerWebhooks(context.Background(), hooks, "t", enum.WebhookTriggerTagCreated, "x")
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.False(t, res[0].Skipped())
	})
	t.Run("other error is returned", func(t *testing.T) {
		st := newSvcExecStore()
		st.listTrigErr = errors.New("db down")
		_, err := svcNewExecutor(t, st, svcURLProvider{}).
			triggerWebhooks(context.Background(), hooks, "t", enum.WebhookTriggerTagCreated, "x")
		require.Error(t, err)
	})
}

func TestSvcTriggerWebhooksFor(t *testing.T) {
	st := newSvcExecStore()
	st.listHooksErr = errors.New("db down")
	ex := svcNewExecutor(t, st, svcURLProvider{})
	_, err := ex.triggerWebhooksFor(context.Background(), nil, "t", enum.WebhookTriggerTagCreated, "x")
	require.Error(t, err)

	st.listHooksErr = nil
	res, err := ex.triggerWebhooksFor(context.Background(), nil, "t", enum.WebhookTriggerTagCreated, "x")
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestSvcExecuteWebhook_Outcomes(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantResult enum.WebhookExecutionResult
		wantErr    bool
	}{
		{"success", 200, enum.WebhookExecutionResultSuccess, false},
		{"server error", 502, enum.WebhookExecutionResultRetriableError, true},
		{"client error", 404, enum.WebhookExecutionResultFatalError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := svcServer(t, tt.status, "resp")
			st := newSvcExecStore()
			ex := svcNewExecutor(t, st, svcURLProvider{})
			wh := &types.WebhookCore{ID: 1, Enabled: true, URL: srv.URL}

			exec, err := ex.executeWebhook(context.Background(), wh, "trig", enum.WebhookTriggerTagCreated, "x", nil)
			assert.Equal(t, tt.wantErr, err != nil)
			require.NotNil(t, exec)
			assert.Equal(t, tt.wantResult, exec.Result)
			require.Len(t, st.created, 1)
			assert.Equal(t, tt.wantResult, st.created[0].Result)
			require.NotNil(t, wh.LatestExecutionResult)
			assert.Equal(t, tt.wantResult, *wh.LatestExecutionResult)
		})
	}
}

func TestSvcExecuteWebhook_ClientVariants(t *testing.T) {
	srv, _ := svcServer(t, http.StatusOK, "")
	tests := []struct {
		name     string
		whType   enum.WebhookType
		insecure bool
	}{
		{"external secure", enum.WebhookTypeExternal, false},
		{"external insecure", enum.WebhookTypeExternal, true},
		{"internal secure", enum.WebhookTypeInternal, false},
		{"internal insecure", enum.WebhookTypeInternal, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
			wh := &types.WebhookCore{ID: 1, URL: srv.URL, Type: tt.whType, Insecure: tt.insecure}
			exec, err := ex.executeWebhook(context.Background(), wh, "t", enum.WebhookTriggerTagCreated, "x", nil)
			require.NoError(t, err)
			assert.Equal(t, enum.WebhookExecutionResultSuccess, exec.Result)
		})
	}
}

func TestSvcExecuteWebhook_BlockedLoopback(t *testing.T) {
	srv, captured := svcServer(t, http.StatusOK, "")
	st := newSvcExecStore()
	ex := NewWebhookExecutor(Config{UserAgentIdentity: "Svctest", HeaderIdentity: "Svctest"},
		svcURLProvider{}, svcEncrypter{}, svcSpacePathStore{}, svcSecretService{}, nil, st, RepoTrigger)
	wh := &types.WebhookCore{ID: 1, URL: srv.URL}

	exec, err := ex.executeWebhook(context.Background(), wh, "t", enum.WebhookTriggerTagCreated, "x", nil)
	require.Error(t, err)
	assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	assert.Equal(t, 0, captured.count, "blocked destination must never be contacted")
}

func TestSvcExecuteWebhook_ConnectionRefused(t *testing.T) {
	srv, _ := svcServer(t, http.StatusOK, "")
	url := srv.URL
	srv.Close()

	st := newSvcExecStore()
	ex := svcNewExecutor(t, st, svcURLProvider{})
	exec, err := ex.executeWebhook(context.Background(),
		&types.WebhookCore{ID: 1, URL: url}, "t", enum.WebhookTriggerTagCreated, "x", nil)
	require.Error(t, err)
	assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	assert.Contains(t, exec.Error, "an error occurred while sending the request")
}

func TestSvcExecuteWebhook_URLProviderError(t *testing.T) {
	st := newSvcExecStore()
	ex := svcNewExecutor(t, st, svcURLProvider{err: errors.New("no url")})
	exec, err := ex.executeWebhook(context.Background(),
		&types.WebhookCore{ID: 1}, "t", enum.WebhookTriggerTagCreated, "x", nil)
	require.Error(t, err)
	assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	require.Len(t, st.created, 1, "execution is stored even if request preparation fails")
}

func TestSvcPrepareHTTPRequest_Body(t *testing.T) {
	ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
	wh := &types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1/hook"}

	t.Run("reader body is used verbatim", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		req, err := ex.prepareHTTPRequest(
			context.Background(), exec, enum.WebhookTriggerTagCreated, wh, bytes.NewBufferString("raw-body"))
		require.NoError(t, err)
		b, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "raw-body", string(b))
		assert.Equal(t, "raw-body", exec.Request.Body)
		assert.Equal(t, wh.URL, exec.Request.URL)
	})
	t.Run("reader failure is fatal", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		_, err := ex.prepareHTTPRequest(context.Background(), exec, enum.WebhookTriggerTagCreated, wh, svcErrReader{})
		require.Error(t, err)
		assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	})
	t.Run("unserializable body is fatal", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		_, err := ex.prepareHTTPRequest(
			context.Background(), exec, enum.WebhookTriggerTagCreated, wh, map[string]any{"c": make(chan int)})
		require.Error(t, err)
		assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	})
	t.Run("invalid url", func(t *testing.T) {
		exec := &types.WebhookExecutionCore{}
		bad := &types.WebhookCore{ID: 1, URL: "http://[::1"}
		_, err := ex.prepareHTTPRequest(context.Background(), exec, enum.WebhookTriggerTagCreated, bad, "x")
		require.Error(t, err)
		assert.Equal(t, enum.WebhookExecutionResultFatalError, exec.Result)
	})
}

func TestSvcPrepareHTTPRequest_HeadersAndSignature(t *testing.T) {
	const body = "payload"
	tests := []struct {
		name       string
		wh         *types.WebhookCore
		encrypter  svcEncrypter
		wantSecret string
		wantErr    bool
	}{
		{"no secret", &types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1"}, svcEncrypter{}, "", false},
		{
			"encrypted secret",
			&types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1", Secret: "mysecret"},
			svcEncrypter{}, "mysecret", false,
		},
		{
			"decrypt failure",
			&types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1", Secret: "mysecret"},
			svcEncrypter{decErr: errors.New("bad")}, "", true,
		},
		{
			"internal secret",
			&types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1", Type: enum.WebhookTypeInternal, Secret: "ignored"},
			svcEncrypter{}, "internalsecret", false,
		},
		{
			"secret manager secret",
			&types.WebhookCore{
				ID: 1, URL: "http://127.0.0.1:1", SecretIdentifier: "sid", SecretSpaceID: 7,
			},
			svcEncrypter{}, "sv:root/space:sid", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
			ex.encrypter = tt.encrypter
			exec := &types.WebhookExecutionCore{}
			req, err := ex.prepareHTTPRequest(context.Background(), exec, enum.WebhookTriggerTagCreated, tt.wh, body)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			sig := req.Header.Get("X-Svctest-Signature")
			if tt.wantSecret == "" {
				assert.Empty(t, sig)
				return
			}
			// body is JSON encoded (with trailing newline) for non-reader bodies.
			want, herr := crypto.GenerateHMACSHA256([]byte("\""+body+"\"\n"), []byte(tt.wantSecret))
			require.NoError(t, herr)
			assert.Equal(t, want, sig)
		})
	}
}

func TestSvcPrepareHTTPRequest_SecretManagerErrors(t *testing.T) {
	wh := &types.WebhookCore{ID: 1, URL: "http://127.0.0.1:1", SecretIdentifier: "sid", SecretSpaceID: 7}

	ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
	ex.spacePathStore = svcSpacePathStore{err: errors.New("no path")}
	_, err := ex.prepareHTTPRequest(
		context.Background(), &types.WebhookExecutionCore{}, enum.WebhookTriggerTagCreated, wh, "x")
	require.Error(t, err)

	ex = svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
	ex.secretService = svcSecretService{err: errors.New("no secret")}
	_, err = ex.prepareHTTPRequest(
		context.Background(), &types.WebhookExecutionCore{}, enum.WebhookTriggerTagCreated, wh, "x")
	require.Error(t, err)
}

func TestSvcPrepareHTTPRequest_ExtraHeadersMasked(t *testing.T) {
	ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})
	wh := &types.WebhookCore{
		ID:  1,
		URL: "http://127.0.0.1:1",
		ExtraHeaders: []types.ExtraHeader{
			{Key: "X-Plain", Value: "plainvalue"},
			{Key: "X-Token", Value: "tokenvalue", Masked: true},
		},
	}
	exec := &types.WebhookExecutionCore{}
	req, err := ex.prepareHTTPRequest(context.Background(), exec, enum.WebhookTriggerTagCreated, wh, "x")
	require.NoError(t, err)

	assert.Equal(t, "plainvalue", req.Header.Get("X-Plain"))
	assert.Equal(t, "tokenvalue", req.Header.Get("X-Token"), "real request carries the real value")
	assert.Contains(t, exec.Request.Headers, "plainvalue")
	assert.NotContains(t, exec.Request.Headers, "tokenvalue", "history must not leak masked values")
	assert.Contains(t, exec.Request.Headers, maskedHeaderValue)
}

func TestSvcRetriggerWebhookExecution(t *testing.T) {
	srv, captured := svcServer(t, http.StatusOK, "")
	st := newSvcExecStore()
	st.webhooks[1] = &types.WebhookCore{ID: 1, Enabled: true, URL: srv.URL}
	st.executions[10] = &types.WebhookExecutionCore{
		ID: 10, WebhookID: 1, TriggerID: "trigger-10", TriggerType: enum.WebhookTriggerTagCreated,
		Retriggerable: true, Request: types.WebhookExecutionRequest{Body: `{"x":1}`},
	}
	st.executions[11] = &types.WebhookExecutionCore{ID: 11, WebhookID: 1, Retriggerable: false}
	st.executions[12] = &types.WebhookExecutionCore{ID: 12, WebhookID: 99, Retriggerable: true}
	ex := svcNewExecutor(t, st, svcURLProvider{})

	t.Run("not found", func(t *testing.T) {
		_, err := ex.RetriggerWebhookExecution(context.Background(), 404)
		require.Error(t, err)
	})
	t.Run("not retriggerable", func(t *testing.T) {
		_, err := ex.RetriggerWebhookExecution(context.Background(), 11)
		require.ErrorIs(t, err, ErrWebhookNotRetriggerable)
	})
	t.Run("webhook missing", func(t *testing.T) {
		_, err := ex.RetriggerWebhookExecution(context.Background(), 12)
		require.Error(t, err)
	})
	t.Run("success reuses trigger id and body", func(t *testing.T) {
		res, err := ex.RetriggerWebhookExecution(context.Background(), 10)
		require.NoError(t, err)
		require.NoError(t, res.Err)
		assert.Equal(t, "trigger-10", res.TriggerID)
		assert.Equal(t, enum.WebhookTriggerTagCreated, res.TriggerType)
		require.NotNil(t, res.Execution.RetriggerOf)
		assert.Equal(t, int64(10), *res.Execution.RetriggerOf)
		assert.JSONEq(t, `{"x":1}`, captured.body)
	})
}

func TestSvcTriggerForEvent(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"success", 200, false},
		{"fatal errors do not retry the event", 404, false},
		{"retriable errors retry the event", 503, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := svcServer(t, tt.status, "")
			st := newSvcExecStore()
			st.webhooks[1] = &types.WebhookCore{ID: 1, Enabled: true, URL: srv.URL}
			ex := svcNewExecutor(t, st, svcURLProvider{})
			err := ex.TriggerForEvent(context.Background(), "evt-1", nil, enum.WebhookTriggerTagCreated, "x")
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

	t.Run("list failure", func(t *testing.T) {
		st := newSvcExecStore()
		st.listHooksErr = errors.New("db down")
		err := svcNewExecutor(t, st, svcURLProvider{}).
			TriggerForEvent(context.Background(), "evt-1", nil, enum.WebhookTriggerTagCreated, "x")
		require.Error(t, err)
	})
}

func TestSvcGenerateTriggerIDFromEventID(t *testing.T) {
	assert.Equal(t, "event-abc", generateTriggerIDFromEventID("abc"))
}

type svcPrincipalStore struct {
	appstore.PrincipalStore
	p   *types.Principal
	err error
}

func (s svcPrincipalStore) Find(context.Context, int64) (*types.Principal, error) { return s.p, s.err }

func TestSvcFindPrincipalForEvent(t *testing.T) {
	ex := svcNewExecutor(t, newSvcExecStore(), svcURLProvider{})

	ex.principalStore = svcPrincipalStore{p: &types.Principal{ID: 5}}
	p, err := ex.FindPrincipalForEvent(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), p.ID)

	ex.principalStore = svcPrincipalStore{err: store.ErrResourceNotFound}
	_, err = ex.FindPrincipalForEvent(context.Background(), 5)
	require.Error(t, err)
	assert.True(t, svcIsDiscard(err))

	ex.principalStore = svcPrincipalStore{err: errors.New("db down")}
	_, err = ex.FindPrincipalForEvent(context.Background(), 5)
	require.Error(t, err)
	assert.False(t, svcIsDiscard(err))
}

func TestSvcCoreConversions(t *testing.T) {
	res := enum.WebhookExecutionResultSuccess
	wh := &types.Webhook{
		ID: 1, Version: 2, ParentID: 3, ParentType: enum.WebhookParentRepo, CreatedBy: 4, Created: 5, Updated: 6,
		Type: enum.WebhookTypeExternal, Scope: 7, Identifier: "id", DisplayName: "dn", Description: "d",
		URL: "http://u", Secret: "s", Enabled: true, Insecure: true,
		Triggers:              []enum.WebhookTrigger{enum.WebhookTriggerTagCreated},
		LatestExecutionResult: &res,
		ExtraHeaders:          []types.ExtraHeader{{Key: "k", Value: "v"}},
	}
	core := GitnessWebhookToWebhookCore(wh)
	back := CoreWebhookToGitnessWebhook(core)
	assert.Equal(t, wh, back)

	list := GitnessWebhooksToWebhooksCore([]*types.Webhook{wh, wh})
	assert.Len(t, list, 2)
	assert.Empty(t, GitnessWebhooksToWebhooksCore(nil))

	rerun := int64(9)
	exec := &types.WebhookExecution{
		ID: 1, WebhookID: 2, TriggerID: "t", TriggerType: enum.WebhookTriggerTagCreated, Result: res, Error: "e",
		Request: types.WebhookExecutionRequest{Body: "b"}, Response: types.WebhookExecutionResponse{Body: "r"},
		RetriggerOf: &rerun, Retriggerable: true, Duration: 8, Created: 10,
	}
	roundTrip := CoreWebhookExecutionToGitnessWebhookExecution(GitnessWebhookExecutionToWebhookExecutionCore(exec))
	assert.Equal(t, exec, roundTrip)
}

func TestSvcGitnessURLProvider(t *testing.T) {
	u, err := NewURLProvider(context.Background()).GetWebhookURL(context.Background(), &types.WebhookCore{URL: "http://x"})
	require.NoError(t, err)
	assert.Equal(t, "http://x", u)
}

func TestSvcGetSecretValue(t *testing.T) {
	v, err := getSecretValue(context.Background(), svcSpacePathStore{}, svcSecretService{value: "s"}, 1, "ident")
	require.NoError(t, err)
	assert.Equal(t, "s:root/space:ident", v)

	_, err = getSecretValue(context.Background(), svcSpacePathStore{err: errors.New("x")}, svcSecretService{}, 1, "i")
	require.Error(t, err)
	_, err = getSecretValue(context.Background(), svcSpacePathStore{}, svcSecretService{err: errors.New("x")}, 1, "i")
	require.Error(t, err)
}
