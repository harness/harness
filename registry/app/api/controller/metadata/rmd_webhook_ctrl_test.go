//  Copyright 2026 Harness, Inc.
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

package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func rmdExternalHook() *coretypes.WebhookCore {
	return &coretypes.WebhookCore{ID: 4, Identifier: "hook", Type: enum.WebhookTypeExternal}
}

func rmdWebhooks(r rmdWebhookRepo) func(*APIController) {
	return func(c *APIController) { c.WebhooksRepository = r }
}

func TestRmdGetWebhook(t *testing.T) {
	req := artifact.GetWebhookRequestObject{RegistryRef: "root/reg", WebhookIdentifier: "hook"}
	// GetWebhook returns server failures as both the response and the error; only the response is checked.
	call := func(c *APIController, ctx context.Context) (any, error) {
		resp, _ := c.GetWebhook(ctx, req)
		return resp, nil
	}
	rmdRunPrelude(t, rmdPrelude{name: "getwebhook", call: call, base: 500, space: 500, anon: 401, forbidden: 403})

	rmdRunCases(t, call, []rmdCase{
		{name: "not_found", want: 404, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errRmdNotFound})},
		{name: "store_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errors.New("db")})},
		{name: "mapping_error", want: 500, mutate: func(c *APIController) {
			rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})(c)
			c.RegistryMetadataHelper = rmdHelper{info: rmdBaseInfo(), respErr: errors.New("map")}
		}},
		{name: "success", want: 200, mutate: rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})},
	})

	resp, err := rmdNewCtrl(rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})).GetWebhook(rmdAdminCtx(), req)
	require.NoError(t, err)
	ok, isOK := resp.(artifact.GetWebhook200JSONResponse)
	require.True(t, isOK)
	require.Equal(t, "hook", ok.Data.Identifier)
}

func TestRmdListWebhooks(t *testing.T) {
	size := artifact.PageSize(5)
	page := artifact.PageNumber(2)
	order := artifact.SortOrder("DESC")
	field := artifact.SortField("name")
	search := artifact.SearchTerm("hook")
	req := artifact.ListWebhooksRequestObject{
		RegistryRef: "root/reg",
		Params: artifact.ListWebhooksParams{
			Size: &size, Page: &page, SortOrder: &order, SortField: &field, SearchTerm: &search,
		},
	}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.ListWebhooks(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "listwebhooks", call: call, base: 500, space: 500, anon: 401, forbidden: 403})

	hooks := []*coretypes.WebhookCore{rmdExternalHook(), {Identifier: "second"}}
	rmdRunCases(t, call, []rmdCase{
		{name: "list_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{listErr: errors.New("db")})},
		{name: "count_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{list: hooks, countErr: errors.New("db")})},
		{name: "mapping_error", want: 500, mutate: func(c *APIController) {
			rmdWebhooks(rmdWebhookRepo{list: hooks, count: 2})(c)
			c.RegistryMetadataHelper = rmdHelper{info: rmdBaseInfo(), respErr: errors.New("map")}
		}},
		{name: "success", want: 200, mutate: rmdWebhooks(rmdWebhookRepo{list: hooks, count: 11})},
	})

	t.Run("response_payload", func(t *testing.T) {
		resp, err := rmdNewCtrl(rmdWebhooks(rmdWebhookRepo{list: hooks, count: 11})).ListWebhooks(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.ListWebhooks200JSONResponse)
		require.True(t, isOK)
		require.Len(t, ok.Data.Webhooks, 2)
		require.Equal(t, int64(3), *ok.Data.PageCount)
		require.Equal(t, int64(11), *ok.Data.ItemCount)
		require.Equal(t, int64(2), *ok.Data.PageIndex)
		require.Equal(t, 5, *ok.Data.PageSize)
	})

	t.Run("defaults_without_params", func(t *testing.T) {
		resp, err := rmdNewCtrl(rmdWebhooks(rmdWebhookRepo{count: 0})).
			ListWebhooks(rmdAdminCtx(), artifact.ListWebhooksRequestObject{RegistryRef: "root/reg"})
		require.NoError(t, err)
		ok, isOK := resp.(artifact.ListWebhooks200JSONResponse)
		require.True(t, isOK)
		require.Empty(t, ok.Data.Webhooks)
		require.Equal(t, 10, *ok.Data.PageSize)
	})
}

func TestRmdUpdateWebhook(t *testing.T) {
	body := artifact.UpdateWebhookJSONRequestBody{Identifier: "hook", Name: "n", Url: "https://hook.example.com"}
	req := artifact.UpdateWebhookRequestObject{RegistryRef: "root/reg", WebhookIdentifier: "hook", Body: &body}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.UpdateWebhook(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "updatewebhook", call: call, base: 500, space: 500, anon: 401, forbidden: 403})

	rmdRunCases(t, call, []rmdCase{
		{name: "not_found", want: 404, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errRmdNotFound})},
		{name: "lookup_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errors.New("db")})},
		{name: "internal_webhook_is_read_only", want: 400, mutate: rmdWebhooks(rmdWebhookRepo{
			byIdent: &coretypes.WebhookCore{Type: enum.WebhookTypeInternal},
		})},
		{name: "mapping_error", want: 400, mutate: func(c *APIController) {
			rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})(c)
			c.RegistryMetadataHelper = rmdHelper{info: rmdBaseInfo(), coreErr: errors.New("map")}
		}},
		{name: "update_error", want: 400, mutate: rmdWebhooks(rmdWebhookRepo{
			byIdent: rmdExternalHook(), updateErr: errors.New("db"),
		})},
		{name: "reload_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{
			byIdent: rmdExternalHook(), secondGetErr: errors.New("db"), calls: new(int),
		})},
		{name: "response_mapping_error", want: 500, mutate: func(c *APIController) {
			rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})(c)
			c.RegistryMetadataHelper = rmdHelper{info: rmdBaseInfo(), respErr: errors.New("map")}
		}},
		{name: "success", want: 201, mutate: rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})},
	})

	t.Run("identifier_comes_from_the_path", func(t *testing.T) {
		var updated *coretypes.WebhookCore
		c := rmdNewCtrl(rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook(), updated: &updated}))
		_, err := c.UpdateWebhook(rmdAdminCtx(), artifact.UpdateWebhookRequestObject{
			RegistryRef: "root/reg", WebhookIdentifier: "path-ident", Body: &body,
		})
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, "path-ident", updated.Identifier)
	})
}

func TestRmdDeleteWebhook(t *testing.T) {
	req := artifact.DeleteWebhookRequestObject{RegistryRef: "root/reg", WebhookIdentifier: "hook"}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.DeleteWebhook(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "deletewebhook", call: call, base: 500, space: 500, anon: 401, forbidden: 403})

	rmdRunCases(t, call, []rmdCase{
		{name: "not_found", want: 404, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errRmdNotFound})},
		{name: "lookup_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errors.New("db")})},
		{name: "internal_webhook_cannot_be_deleted", want: 400, mutate: rmdWebhooks(rmdWebhookRepo{
			byIdent: &coretypes.WebhookCore{Type: enum.WebhookTypeInternal},
		})},
		{name: "delete_error", want: 500, mutate: rmdWebhooks(rmdWebhookRepo{
			byIdent: rmdExternalHook(), deleteErr: errors.New("db"),
		})},
		{name: "success", want: 200, mutate: rmdWebhooks(rmdWebhookRepo{byIdent: rmdExternalHook()})},
	})
}

func TestRmdCreateWebhookValidation(t *testing.T) {
	valid := func() artifact.CreateWebhookJSONRequestBody {
		return artifact.CreateWebhookJSONRequestBody{
			Identifier: "hook", Name: "name", Url: "https://hook.example.com/path",
		}
	}
	setURL := func(u string) func(b *artifact.CreateWebhookJSONRequestBody) {
		return func(b *artifact.CreateWebhookJSONRequestBody) { b.Url = u }
	}
	setIdentifier := func(id string) func(b *artifact.CreateWebhookJSONRequestBody) {
		return func(b *artifact.CreateWebhookJSONRequestBody) { b.Identifier = id }
	}
	virtualInfo := func() *types.RegistryRequestBaseInfo {
		info := rmdBaseInfo()
		info.RegistryType = artifact.RegistryTypeVIRTUAL
		return info
	}
	run := func(body artifact.CreateWebhookJSONRequestBody, mutate func(*APIController), ctx context.Context) int {
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryMetadataHelper = rmdHelper{info: virtualInfo()}
			c.WebhooksRepository = rmdWebhookRepo{byIdent: rmdExternalHook()}
			if mutate != nil {
				mutate(c)
			}
		})
		resp, err := c.CreateWebhook(ctx, artifact.CreateWebhookRequestObject{RegistryRef: "root/reg", Body: &body})
		require.NoError(t, err)
		return rmdStatus(t, resp)
	}

	tests := []struct {
		name   string
		mod    func(b *artifact.CreateWebhookJSONRequestBody)
		mutate func(*APIController)
		ctx    context.Context
		want   int
	}{
		{name: "valid", mod: func(*artifact.CreateWebhookJSONRequestBody) {}, want: 201},
		{name: "missing_identifier", mod: setIdentifier(""), want: 400},
		{name: "missing_name", mod: func(b *artifact.CreateWebhookJSONRequestBody) { b.Name = "" }, want: 400},
		{name: "missing_url", mod: setURL(""), want: 400},
		{name: "unparseable_url", mod: setURL("http://[::1"), want: 400},
		{name: "url_without_host", mod: setURL("https:///path"), want: 400},
		{name: "url_without_scheme", mod: setURL("hook.example.com"), want: 400},
		{name: "ftp_scheme_rejected", mod: setURL("ftp://hook.example.com"), want: 400},
		{name: "uppercase_scheme_allowed", mod: setURL("HTTPS://hook.example.com"), want: 201},
		{name: "reserved_identifier", mod: setIdentifier(internalWebhookIdentifier),
			want: 400},
		{name: "base_info_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("x")} }, want: 400},
		{name: "upstream_registry_not_allowed", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{info: rmdBaseInfo()} }, want: 400},
		{name: "space_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} }, want: 400},
		{name: "anonymous_denied", mod: func(*artifact.CreateWebhookJSONRequestBody) {}, ctx: rmdAnonCtx(),
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} }, want: 401},
		{name: "forbidden", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} }, want: 403},
		{name: "mapping_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) {
				c.RegistryMetadataHelper = rmdHelper{info: virtualInfo(), coreErr: errors.New("x")}
			}, want: 400},
		{name: "duplicate_identifier", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: rmdWebhooks(rmdWebhookRepo{createErr: errors.New("resource is a duplicate")}), want: 400},
		{name: "create_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: rmdWebhooks(rmdWebhookRepo{createErr: errors.New("db")}), want: 400},
		{name: "reload_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: rmdWebhooks(rmdWebhookRepo{byIdentErr: errors.New("db")}), want: 500},
		{name: "response_mapping_error", mod: func(*artifact.CreateWebhookJSONRequestBody) {},
			mutate: func(c *APIController) {
				c.RegistryMetadataHelper = rmdHelper{info: virtualInfo(), respErr: errors.New("x")}
			}, want: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := valid()
			tt.mod(&body)
			ctx := tt.ctx
			if ctx == nil {
				ctx = rmdAdminCtx()
			}
			require.Equal(t, tt.want, run(body, tt.mutate, ctx))
		})
	}
}
