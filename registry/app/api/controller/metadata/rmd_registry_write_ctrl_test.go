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
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func rmdCreateBody(r *artifact.RegistryRequest) *artifact.CreateRegistryJSONRequestBody {
	return (*artifact.CreateRegistryJSONRequestBody)(r)
}

func rmdModifyBody(r *artifact.RegistryRequest) *artifact.ModifyRegistryJSONRequestBody {
	return (*artifact.ModifyRegistryJSONRequestBody)(r)
}

func rmdVirtualBody(t *testing.T, id string, upstreams ...string) *artifact.RegistryRequest {
	t.Helper()
	rc := &artifact.RegistryConfig{}
	require.NoError(t, rc.FromVirtualConfig(artifact.VirtualConfig{UpstreamProxies: &upstreams}))
	return &artifact.RegistryRequest{
		Identifier: id, PackageType: artifact.PackageTypeNPM, ParentRef: "root", Config: rc,
		CleanupPolicy: &[]artifact.CleanupPolicy{{
			Name: rmdStrPtr("p"), ExpireDays: func() *int { d := 1; return &d }(),
			VersionPrefix: &[]string{}, PackagePrefix: &[]string{},
		}},
	}
}

func rmdUpstreamBody(t *testing.T, id string, cfg artifact.UpstreamConfig) *artifact.RegistryRequest {
	t.Helper()
	return &artifact.RegistryRequest{
		Identifier: id, PackageType: artifact.PackageTypeNPM, ParentRef: "root",
		Config: rmdUpstreamConfigBody(t, cfg),
	}
}

func rmdAnonymousUpstream() artifact.UpstreamConfig {
	src := artifact.UpstreamConfigSourceCustom
	return artifact.UpstreamConfig{
		AuthType: artifact.AuthTypeAnonymous, Source: &src, Url: rmdStrPtr("https://up.example.com"),
	}
}

func TestRmdCreateRegistry(t *testing.T) {
	call := func(c *APIController, ctx context.Context, r artifact.CreateRegistryRequestObject) (
		artifact.CreateRegistryResponseObject, error,
	) {
		return c.CreateRegistry(ctx, r)
	}
	virtualReq := artifact.CreateRegistryRequestObject{Body: rmdCreateBody(rmdVirtualBody(t, "virt-reg"))}
	publicVirtual := rmdVirtualBody(t, "virt-reg")
	publicVirtual.IsPublic = true
	upstreamReq := artifact.CreateRegistryRequestObject{
		Body: rmdCreateBody(rmdUpstreamBody(t, "up-reg", rmdAnonymousUpstream())),
	}

	created := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{createID: 5, byID: map[int64]*types.Registry{
			5: {ID: 5, UUID: "u", Name: "virt-reg", ParentID: 2, Type: artifact.RegistryTypeVIRTUAL},
		}}
	}
	upstreamCreated := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{createID: 5}
		c.UpstreamProxyStore = rmdUpstreamRepo{byID: rmdUpstreamEntity()}
	}

	t.Run("common_failures", func(t *testing.T) {
		rmdRunFailureCases(t, call, virtualReq, []rmdFailureCase{
			{
				name:   "base_info_error",
				mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("x")} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{
				name:   "space_error",
				mutate: func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{
				name:   "anonymous_denied",
				ctx:    rmdAnonCtx(),
				mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
				want:   artifact.CreateRegistry401JSONResponse{},
			},
			{
				name:   "forbidden",
				mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
				want:   artifact.CreateRegistry403JSONResponse{},
			},
		})
	})

	t.Run("public_registry_checks", func(t *testing.T) {
		publicReq := artifact.CreateRegistryRequestObject{Body: rmdCreateBody(publicVirtual)}
		rmdRunFailureCases(t, call, publicReq, []rmdFailureCase{
			{
				name:   "support_check_error",
				mutate: func(c *APIController) { c.PublicAccess = rmdPublicAccess{supportedErr: errors.New("x")} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{
				name:   "public_access_not_supported",
				mutate: func(c *APIController) { c.PublicAccess = rmdPublicAccess{supported: false} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{name: "public_virtual_created", mutate: created, want: artifact.CreateRegistry201JSONResponse{}},
		})
	})

	t.Run("virtual_registry", func(t *testing.T) {
		rmdRunFailureCases(t, call, virtualReq, []rmdFailureCase{
			{
				name:   "invalid_package_type",
				mutate: func(c *APIController) { c.PackageWrapper = rmdPkgWrapper{invalidPkgType: true} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "create_error",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createErr: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "duplicate_resolved_to_friendly_message",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createErr: errors.New("resource is a duplicate")}
					c.RegFinder = rmdRegFinder{found: &types.Registry{ParentID: 2}}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "duplicate_lookup_fails",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createErr: errors.New("resource is a duplicate")}
					c.RegFinder = rmdRegFinder{findErr: errors.New("x")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "public_access_set_fails_and_cleanup_succeeds",
				mutate: func(c *APIController) {
					created(c)
					c.PublicAccess = rmdPublicAccess{supported: true, setErr: errors.New("pa")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "reload_error",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createID: 5, getErr: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "cleanup_policy_error",
				mutate: func(c *APIController) {
					created(c)
					c.CleanupPolicyStore = rmdCleanupRepo{getErr: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "reload_space_error",
				mutate: func(c *APIController) {
					created(c)
					c.SpaceFinder = &rmdSeqSpaceFinder{results: []rmdSpaceResult{
						{space: rmdSpaceCore()}, {err: errors.New("x")},
					}}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "response_build_error",
				mutate: func(c *APIController) {
					created(c)
					c.PublicAccess = rmdPublicAccess{supported: true, getErr: errors.New("pa")}
				},
				want: artifact.CreateRegistry500JSONResponse{},
			},
			{name: "success", mutate: created, want: artifact.CreateRegistry201JSONResponse{}},
		})
	})

	t.Run("virtual_registry_with_upstreams", func(t *testing.T) {
		req := artifact.CreateRegistryRequestObject{Body: rmdCreateBody(rmdVirtualBody(t, "virt-reg", "up1"))}
		metas := &[]store.RegistryMetadata{{RegIdentifier: "up1", RegID: "11"}}
		rmdRunFailureCases(t, call, req, []rmdFailureCase{
			{
				name: "upstream_resolution_error",
				mutate: func(c *APIController) {
					c.SpaceStore = rmdSpaceStore{err: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "success",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createID: 5, all: metas, byID: map[int64]*types.Registry{
						5:  {ID: 5, Name: "virt-reg", ParentID: 2, UpstreamProxies: []int64{11}},
						11: {ID: 11},
					}}
				},
				want: artifact.CreateRegistry201JSONResponse{},
			},
		})
	})

	t.Run("generic_virtual_skips_upstream_resolution", func(t *testing.T) {
		body := rmdVirtualBody(t, "virt-reg", "up1")
		body.PackageType = artifact.PackageTypeGENERIC
		c := rmdNewCtrl(created, func(c *APIController) {
			c.SpaceStore = rmdSpaceStore{err: errors.New("must not be called")}
		})
		resp, err := c.CreateRegistry(rmdAdminCtx(), artifact.CreateRegistryRequestObject{Body: rmdCreateBody(body)})
		require.NoError(t, err)
		require.IsType(t, artifact.CreateRegistry201JSONResponse{}, resp)
	})

	t.Run("upstream_registry", func(t *testing.T) {
		secretCfg := artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpaceId: rmdInt64Ptr(4),
			}),
		}
		secretReq := artifact.CreateRegistryRequestObject{Body: rmdCreateBody(rmdUpstreamBody(t, "up-reg", secretCfg))}
		rmdRunFailureCases(t, call, secretReq, []rmdFailureCase{
			{
				name:   "secret_access_forbidden",
				mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: true, denySecret: true} },
				want:   artifact.CreateRegistry403JSONResponse{},
			},
			{
				name:   "secret_access_unauthenticated",
				ctx:    rmdAnonCtx(),
				mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: true, denySecret: true} },
				want:   artifact.CreateRegistry401JSONResponse{},
			},
			{name: "secret_access_granted", mutate: upstreamCreated, want: artifact.CreateRegistry201JSONResponse{}},
		})

		rmdRunFailureCases(t, call, upstreamReq, []rmdFailureCase{
			{
				name:   "invalid_entity",
				mutate: func(c *APIController) { c.PackageWrapper = rmdPkgWrapper{invalidPkgType: true} },
				want:   artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "tx_error",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createID: 5}
					c.tx = rmdTx{err: errors.New("tx")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "tx_duplicate_error",
				mutate: func(c *APIController) {
					c.tx = rmdTx{err: errors.New("resource is a duplicate")}
					c.RegFinder = rmdRegFinder{found: &types.Registry{ParentID: 2}}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "upstream_create_error",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createID: 5}
					c.UpstreamProxyStore = rmdUpstreamRepo{createErr: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "upstream_reload_error",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{createID: 5}
					c.UpstreamProxyStore = rmdUpstreamRepo{byIDErr: errors.New("db")}
				},
				want: artifact.CreateRegistry400JSONResponse{},
			},
			{
				name: "response_build_error",
				mutate: func(c *APIController) {
					upstreamCreated(c)
					c.PublicAccess = rmdPublicAccess{supported: true, getErr: errors.New("pa")}
				},
				want: artifact.CreateRegistry500JSONResponse{},
			},
			{name: "success", mutate: upstreamCreated, want: artifact.CreateRegistry201JSONResponse{}},
		})
	})
}

func TestRmdCreateRegistryHelpers(t *testing.T) {
	ctx := rmdAdminCtx()
	principal := &rmdPrincipal

	t.Run("public_access_failure_rolls_back", func(t *testing.T) {
		repo := &rmdRegRepo{createID: 5}
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = repo
			c.PublicAccess = rmdPublicAccess{setErr: errors.New("pa")}
		})
		id, err := c.createRegistry(ctx, &types.Registry{Name: "r", ParentID: 2}, "root", principal, false)
		require.Zero(t, id)
		require.ErrorContains(t, err, "successful cleanup")
		require.Equal(t, 1, repo.deleteCalls)
	})

	t.Run("rollback_reports_cleanup_failures", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.PublicAccess = rmdPublicAccess{setErr: errors.New("pa"), deleteErr: errors.New("pa-del")}
		})
		_, err := c.createRegistry(ctx, &types.Registry{Name: "r"}, "root", principal, false)
		require.ErrorContains(t, err, "public access cleanup")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{deleteErr: errors.New("reg-del")}
			c.PublicAccess = rmdPublicAccess{setErr: errors.New("pa")}
		})
		_, err = c.createRegistry(ctx, &types.Registry{Name: "r"}, "root", principal, false)
		require.ErrorContains(t, err, "registry delete")
	})

	t.Run("audit_written_unless_skipped", func(t *testing.T) {
		var entries []string
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{createID: 5, byID: map[int64]*types.Registry{5: {UUID: "fresh"}}}
			c.AuditService = rmdAudit{entries: &entries}
		})
		reg := &types.Registry{Name: "r"}
		id, err := c.createRegistry(ctx, reg, "root", principal, false)
		require.NoError(t, err)
		require.Equal(t, int64(5), id)
		require.Equal(t, "fresh", reg.UUID)
		require.Equal(t, []string{"created:r"}, entries)

		_, err = c.createRegistry(ctx, &types.Registry{Name: "r2"}, "root", principal, true)
		require.NoError(t, err)
		_, err = c.createRegistry(ctx, &types.Registry{Name: "r3"}, "root", nil, false)
		require.NoError(t, err)
		require.Len(t, entries, 1)
	})

	t.Run("get_after_create_failure_is_tolerated", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{createID: 5, getErr: errors.New("db")}
			c.AuditService = rmdAudit{err: errors.New("audit")}
		})
		id, err := c.createRegistry(ctx, &types.Registry{Name: "r"}, "root", principal, false)
		require.NoError(t, err)
		require.Equal(t, int64(5), id)
	})

	t.Run("upstream_proxy_audit", func(t *testing.T) {
		var entries []string
		c := rmdNewCtrl(func(c *APIController) { c.AuditService = rmdAudit{entries: &entries} })
		id, err := c.createUpstreamProxyWithAudit(ctx, &types.Registry{Name: "r"},
			&types.UpstreamProxyConfig{}, rmdPrincipal, "root")
		require.NoError(t, err)
		require.Equal(t, int64(1), id)
		require.Len(t, entries, 1)

		failing := rmdNewCtrl(func(c *APIController) {
			c.UpstreamProxyStore = rmdUpstreamRepo{createErr: errors.New("db")}
		})
		_, err = failing.createUpstreamProxyWithAudit(ctx, &types.Registry{}, &types.UpstreamProxyConfig{},
			rmdPrincipal, "root")
		require.Error(t, err)

		auditFail := rmdNewCtrl(func(c *APIController) { c.AuditService = rmdAudit{err: errors.New("audit")} })
		_, err = auditFail.createUpstreamProxyWithAudit(ctx, &types.Registry{}, &types.UpstreamProxyConfig{},
			rmdPrincipal, "root")
		require.NoError(t, err)
	})

	t.Run("is_duplicate_key_error", func(t *testing.T) {
		require.True(t, isDuplicateKeyError(errors.New("x: resource is a duplicate")))
		require.False(t, isDuplicateKeyError(errors.New("other")))
	})

	t.Run("duplicate_error_messages", func(t *testing.T) {
		reg := &types.Registry{Name: "dup", RootParentID: 1}
		ok := rmdNewCtrl(func(c *APIController) { c.RegFinder = rmdRegFinder{found: &types.Registry{ParentID: 2}} })
		err := ok.handleDuplicateRegistryError(ctx, reg)
		require.ErrorContains(t, err, "registry 'dup' is already defined under 'root'")

		noSpace := rmdNewCtrl(func(c *APIController) {
			c.RegFinder = rmdRegFinder{found: &types.Registry{ParentID: 2}}
			c.SpaceFinder = rmdSpaceFinder{}
		})
		require.ErrorContains(t, noSpace.handleDuplicateRegistryError(ctx, reg), "parent space not found")

		spaceErr := rmdNewCtrl(func(c *APIController) {
			c.RegFinder = rmdRegFinder{found: &types.Registry{ParentID: 2}}
			c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")}
		})
		require.ErrorContains(t, spaceErr.handleDuplicateRegistryError(ctx, reg), "fetch space details")
	})
}

func TestRmdModifyRegistry(t *testing.T) {
	call := func(c *APIController, ctx context.Context, r artifact.ModifyRegistryRequestObject) (
		artifact.ModifyRegistryResponseObject, error,
	) {
		return c.ModifyRegistry(ctx, r)
	}
	virtualReq := artifact.ModifyRegistryRequestObject{
		RegistryRef: "root/reg", Body: rmdModifyBody(rmdVirtualBody(t, "reg")),
	}
	upstreamReq := artifact.ModifyRegistryRequestObject{
		RegistryRef: "root/reg", Body: rmdModifyBody(rmdUpstreamBody(t, "reg", rmdAnonymousUpstream())),
	}
	virtual := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{byParentName: rmdVirtualRepoEntity()}
	}
	upstreamEntity := rmdUpstreamEntity()
	upstreamEntity.Source = "Custom"
	upstreamEntity.RepoURL = "https://up.example.com"
	upstream := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{byParentName: &types.Registry{
			ID: 5, Name: "reg", Type: artifact.RegistryTypeUPSTREAM,
		}}
		c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: upstreamEntity, byID: upstreamEntity}
	}

	rmdRunFailureCases(t, call, virtualReq, []rmdFailureCase{
		{
			name:   "base_info_error",
			mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("x")} },
			want:   artifact.ModifyRegistry400JSONResponse{},
		},
		{
			name:   "space_error",
			mutate: func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} },
			want:   artifact.ModifyRegistry400JSONResponse{},
		},
		{
			name:   "anonymous_denied",
			ctx:    rmdAnonCtx(),
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.ModifyRegistry401JSONResponse{},
		},
		{
			name:   "forbidden",
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.ModifyRegistry403JSONResponse{},
		},
		{
			name: "virtual_without_name_is_not_found",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentName: &types.Registry{Type: artifact.RegistryTypeVIRTUAL}}
			},
			want: artifact.ModifyRegistry404JSONResponse{},
		},
		{
			name: "virtual_package_type_change_rejected",
			mutate: func(c *APIController) {
				e := rmdVirtualRepoEntity()
				e.PackageType = artifact.PackageTypeDOCKER
				c.RegistryRepository = &rmdRegRepo{byParentName: e}
			},
			want: artifact.ModifyRegistry400JSONResponse{},
		},
		{
			name: "virtual_public_access_not_supported",
			mutate: func(c *APIController) {
				virtual(c)
				c.PublicAccess = rmdPublicAccess{supported: false}
				c.RegFinder = rmdRegFinder{}
			},
			want: artifact.ModifyRegistry200JSONResponse{},
		},
		{
			name: "virtual_update_failure",
			mutate: func(c *APIController) {
				virtual(c)
				c.RegFinder = rmdRegFinder{updateErr: errors.New("db")}
			},
			want: artifact.ModifyRegistry500JSONResponse{},
		},
		{
			name: "virtual_cleanup_policy_invalid",
			mutate: func(c *APIController) {
				virtual(c)
				c.CleanupPolicyStore = rmdCleanupRepo{idsErr: errors.New("db")}
			},
			want: artifact.ModifyRegistry400JSONResponse{},
		},
		{
			name: "virtual_reload_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentName: rmdVirtualRepoEntity(), getErr: errors.New("db")}
			},
			want: artifact.ModifyRegistry500JSONResponse{},
		},
		{
			name: "virtual_cleanup_policy_reload_error",
			mutate: func(c *APIController) {
				virtual(c)
				c.CleanupPolicyStore = rmdCleanupRepo{getErr: errors.New("db")}
			},
			want: artifact.ModifyRegistry500JSONResponse{},
		},
		{
			name: "virtual_public_access_error",
			mutate: func(c *APIController) {
				virtual(c)
				c.PublicAccess = rmdPublicAccess{supported: true, getErr: errors.New("pa")}
			},
			want: artifact.ModifyRegistry500JSONResponse{},
		},
		{name: "virtual_success", mutate: virtual, want: artifact.ModifyRegistry200JSONResponse{}},
	})

	secretBody := rmdUpstreamBody(t, "reg", artifact.UpstreamConfig{
		AuthType: artifact.AuthTypeUserPassword,
		Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
			UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpaceId: rmdInt64Ptr(4),
		}),
	})
	renamed := rmdUpstreamBody(t, "other", rmdAnonymousUpstream())

	rmdRunFailureCases(t, call, upstreamReq, []rmdFailureCase{
		{
			name: "missing_upstream_record_is_not_found",
			mutate: func(c *APIController) {
				upstream(c)
				c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: &types.UpstreamProxy{}}
			},
			want: artifact.ModifyRegistry404JSONResponse{},
		},
		{name: "success", mutate: upstream, want: artifact.ModifyRegistry200JSONResponse{}},
	})

	t.Run("upstream_invalid_update_is_bad_request", func(t *testing.T) {
		resp, err := rmdNewCtrl(upstream).ModifyRegistry(rmdAdminCtx(),
			artifact.ModifyRegistryRequestObject{RegistryRef: "root/reg", Body: rmdModifyBody(renamed)})
		require.NoError(t, err)
		require.IsType(t, artifact.ModifyRegistry400JSONResponse{}, resp)
	})

	t.Run("upstream_secret_change_requires_secret_access", func(t *testing.T) {
		r := artifact.ModifyRegistryRequestObject{RegistryRef: "root/reg", Body: rmdModifyBody(secretBody)}
		deny := func(c *APIController) {
			upstream(c)
			c.Authorizer = rmdAuthorizer{allow: true, denySecret: true}
		}
		resp, err := rmdNewCtrl(deny).ModifyRegistry(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.ModifyRegistry403JSONResponse{}, resp)

		resp, err = rmdNewCtrl(deny).ModifyRegistry(rmdAnonCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.ModifyRegistry401JSONResponse{}, resp)

		resp, err = rmdNewCtrl(upstream).ModifyRegistry(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.ModifyRegistry200JSONResponse{}, resp)
	})

	t.Run("store_failures_are_returned_as_errors", func(t *testing.T) {
		failures := map[string]func(c *APIController){
			"registry_lookup": func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
			},
			"upstream_lookup": func(c *APIController) {
				upstream(c)
				c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: upstreamEntity, byIdentErr: errors.New("db")}
			},
			"transaction": func(c *APIController) {
				upstream(c)
				c.tx = rmdTx{err: errors.New("tx")}
			},
			"reload": func(c *APIController) {
				upstream(c)
				c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: upstreamEntity, byIDErr: errors.New("db")}
			},
		}
		for name, mutate := range failures {
			t.Run(name, func(t *testing.T) {
				resp, err := rmdNewCtrl(mutate).ModifyRegistry(rmdAdminCtx(), upstreamReq)
				require.Error(t, err)
				require.IsType(t, artifact.ModifyRegistry500JSONResponse{}, resp)
			})
		}
	})
}

func TestRmdUpdatePublicAccess(t *testing.T) {
	ctx := context.Background()
	virtualOld := &types.Registry{Type: artifact.RegistryTypeVIRTUAL, UpstreamProxies: []int64{1}}
	newReg := func(public bool, upstreams ...int64) *types.Registry {
		return &types.Registry{
			ID: 5, Name: "reg", RootParentID: 1, Type: artifact.RegistryTypeVIRTUAL, IsPublic: public,
			UpstreamProxies: upstreams,
		}
	}
	upstreamUsers := &[]types.Registry{{Name: "pub-user", ParentID: 2}}

	t.Run("failures", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(c *APIController)
			reg     *types.Registry
			wantErr string
		}{
			{
				name:    "space_lookup",
				mutate:  func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("space")} },
				reg:     newReg(false),
				wantErr: "space",
			},
			{
				name:    "support_check",
				mutate:  func(c *APIController) { c.PublicAccess = rmdPublicAccess{supportedErr: errors.New("x")} },
				reg:     newReg(false),
				wantErr: "failed to check if public access is supported",
			},
			{
				name:    "public_not_supported",
				mutate:  func(c *APIController) { c.PublicAccess = rmdPublicAccess{supported: false} },
				reg:     newReg(true),
				wantErr: "Public access to artifact registries is disabled",
			},
			{
				name: "current_status_lookup",
				mutate: func(c *APIController) {
					c.PublicAccess = rmdPublicAccess{supported: true, getErr: errors.New("x")}
				},
				reg:     newReg(false),
				wantErr: "current public access status",
			},
			{
				name: "set_failure",
				mutate: func(c *APIController) {
					c.PublicAccess = rmdPublicAccess{supported: true, isPublic: false, setErr: errors.New("x")}
				},
				reg:     newReg(true),
				wantErr: "public access",
			},
			{
				name: "private_upstream_for_public_virtual",
				mutate: func(c *APIController) {
					c.PublicAccess = rmdPublicAccess{supported: true, isPublic: true}
					c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{{Name: "priv", ParentID: 2}}}
					c.PublicAccess = rmdPublicAccess{
						supported: true, isPublic: true, publicRefs: map[string]bool{"root/priv": false},
					}
				},
				reg:     newReg(true, 2),
				wantErr: "private registries upstreams",
			},
			{
				name: "going_private_while_used_by_public_registry",
				mutate: func(c *APIController) {
					c.RegistryRepository = &rmdRegRepo{usedByIDs: []int64{9}, byIDIn: upstreamUsers}
					c.PublicAccess = rmdPublicAccess{
						supported: true, isPublic: true, publicRefs: map[string]bool{"root/pub-user": true},
					}
				},
				reg:     newReg(false),
				wantErr: "is being used inside public registry",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := rmdNewCtrl(tt.mutate)
				err := c.updatePublicAccess(ctx, "root", tt.reg, virtualOld)
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("successes", func(t *testing.T) {
		var sets []string
		c := rmdNewCtrl(func(c *APIController) {
			c.PublicAccess = rmdPublicAccess{supported: true, isPublic: false, setCalls: &sets}
			c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{{Name: "pubup", ParentID: 2}}}
		})
		require.NoError(t, c.updatePublicAccess(ctx, "root", newReg(false), virtualOld), "no change")
		require.Empty(t, sets)

		pub := rmdNewCtrl(func(c *APIController) {
			c.PublicAccess = rmdPublicAccess{
				supported: true, isPublic: false, setCalls: &sets, publicRefs: map[string]bool{"root/reg": false},
			}
			c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{}}
		})
		require.NoError(t, pub.updatePublicAccess(ctx, "root", newReg(true), virtualOld))
		require.Equal(t, []string{"root/reg"}, sets)

		sets = nil
		priv := rmdNewCtrl(func(c *APIController) {
			c.PublicAccess = rmdPublicAccess{
				supported: true, isPublic: true, setCalls: &sets, publicRefs: map[string]bool{"root/reg": true},
			}
			c.RegistryRepository = &rmdRegRepo{}
		})
		require.NoError(t, priv.updatePublicAccess(ctx, "root", newReg(false), virtualOld))
		require.Equal(t, []string{"root/reg"}, sets)
	})
}

func TestRmdUpstreamUsageChecks(t *testing.T) {
	ctx := context.Background()

	t.Run("used_in_public_registries_errors", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{usedByErr: errors.New("db")}
		})
		require.ErrorContains(t, c.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5), "fetch registryIDs")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{usedByIDs: []int64{1}, byIDInErr: errors.New("db")}
		})
		require.ErrorContains(t, c.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5), "fetch registries")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{usedByIDs: []int64{1}, byIDIn: &[]types.Registry{{Name: "x"}}}
			c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")}
		})
		require.ErrorContains(t, c.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5), "space details")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{usedByIDs: []int64{1}, byIDIn: &[]types.Registry{{Name: "x"}}}
			c.PublicAccess = rmdPublicAccess{getErr: errors.New("x")}
		})
		require.ErrorContains(t, c.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5), "public access")
	})

	t.Run("used_only_by_private_registries_is_allowed", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{usedByIDs: []int64{1}, byIDIn: &[]types.Registry{{Name: "x"}}}
			c.PublicAccess = rmdPublicAccess{isPublic: false}
		})
		require.NoError(t, c.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5))
		none := rmdNewCtrl()
		require.NoError(t, none.checkIfUpstreamIsUsedInPublicRegistries(ctx, 1, "up", 5))
	})

	t.Run("virtual_private_upstream_errors", func(t *testing.T) {
		require.NoError(t, rmdNewCtrl().checkIfVirtualHasPrivateUpstreams(ctx, "virt", nil))

		c := rmdNewCtrl(func(c *APIController) { c.RegistryRepository = &rmdRegRepo{byIDInErr: errors.New("db")} })
		require.ErrorContains(t, c.checkIfVirtualHasPrivateUpstreams(ctx, "virt", []int64{1}), "fetch registries")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{{Name: "x"}}}
			c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")}
		})
		require.ErrorContains(t, c.checkIfVirtualHasPrivateUpstreams(ctx, "virt", []int64{1}), "space details")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{{Name: "x"}}}
			c.PublicAccess = rmdPublicAccess{getErr: errors.New("x")}
		})
		require.ErrorContains(t, c.checkIfVirtualHasPrivateUpstreams(ctx, "virt", []int64{1}), "public access")

		c = rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byIDIn: &[]types.Registry{{Name: "x"}}}
			c.PublicAccess = rmdPublicAccess{isPublic: true}
		})
		require.NoError(t, c.checkIfVirtualHasPrivateUpstreams(ctx, "virt", []int64{1}))
	})
}

func TestRmdUpdateRegistryAuditHelpers(t *testing.T) {
	ctx := rmdAdminCtx()
	t.Run("upstream_proxy_update_audited", func(t *testing.T) {
		var entries []string
		c := rmdNewCtrl(func(c *APIController) {
			c.UpstreamProxyStore = rmdUpstreamRepo{byID: rmdUpstreamEntity()}
			c.AuditService = rmdAudit{entries: &entries}
		})
		err := c.updateUpstreamProxyWithAudit(ctx, &types.UpstreamProxyConfig{RegistryID: 5}, rmdPrincipal, "root", "reg")
		require.NoError(t, err)
		require.Equal(t, []string{"updated:reg"}, entries)
	})

	t.Run("missing_previous_config_skips_audit", func(t *testing.T) {
		var entries []string
		c := rmdNewCtrl(func(c *APIController) {
			c.UpstreamProxyStore = rmdUpstreamRepo{byIDErr: errors.New("gone")}
			c.AuditService = rmdAudit{entries: &entries}
		})
		require.NoError(t, c.updateUpstreamProxyWithAudit(ctx, &types.UpstreamProxyConfig{}, rmdPrincipal, "root", "reg"))
		require.Empty(t, entries)
	})

	t.Run("update_error_and_audit_error", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.UpstreamProxyStore = rmdUpstreamRepo{byID: rmdUpstreamEntity(), updateErr: errors.New("db")}
		})
		require.Error(t, c.updateUpstreamProxyWithAudit(ctx, &types.UpstreamProxyConfig{}, rmdPrincipal, "root", "reg"))

		failingAudit := rmdNewCtrl(func(c *APIController) {
			c.UpstreamProxyStore = rmdUpstreamRepo{byID: rmdUpstreamEntity()}
			c.AuditService = rmdAudit{err: errors.New("audit")}
		})
		require.NoError(t, failingAudit.updateUpstreamProxyWithAudit(
			ctx, &types.UpstreamProxyConfig{}, rmdPrincipal, "root", "reg"))
	})

	t.Run("registry_update_audited", func(t *testing.T) {
		var entries []string
		c := rmdNewCtrl(func(c *APIController) { c.AuditService = rmdAudit{entries: &entries} })
		old := &types.Registry{Name: "reg"}
		require.NoError(t, c.updateRegistryWithAudit(ctx, old, &types.Registry{Name: "reg"}, rmdPrincipal, "root"))
		require.Equal(t, []string{"updated:reg"}, entries)

		finderErr := rmdNewCtrl(func(c *APIController) { c.RegFinder = rmdRegFinder{updateErr: errors.New("x")} })
		require.Error(t, finderErr.updateRegistryWithAudit(ctx, old, &types.Registry{Name: "reg"}, rmdPrincipal, "root"))

		auditErr := rmdNewCtrl(func(c *APIController) { c.AuditService = rmdAudit{err: errors.New("x")} })
		require.NoError(t, auditErr.updateRegistryWithAudit(ctx, old, &types.Registry{Name: "reg"}, rmdPrincipal, "root"))
	})

	t.Run("cleanup_policy_update", func(t *testing.T) {
		body := rmdVirtualBody(t, "reg")
		require.NoError(t, rmdNewCtrl().updateCleanupPolicy(ctx, rmdModifyBody(body), 5))

		idsErr := rmdNewCtrl(func(c *APIController) { c.CleanupPolicyStore = rmdCleanupRepo{idsErr: errors.New("x")} })
		require.Error(t, idsErr.updateCleanupPolicy(ctx, rmdModifyBody(body), 5))

		modifyErr := rmdNewCtrl(func(c *APIController) { c.CleanupPolicyStore = rmdCleanupRepo{modifyErr: errors.New("x")} })
		require.Error(t, modifyErr.updateCleanupPolicy(ctx, rmdModifyBody(body), 5))

		invalid := rmdVirtualBody(t, "reg")
		(*invalid.CleanupPolicy)[0].Name = nil
		require.Error(t, rmdNewCtrl().updateCleanupPolicy(ctx, rmdModifyBody(invalid), 5))
	})

	t.Run("error_helpers", func(t *testing.T) {
		require.Equal(t, "boom", throwModifyRegistry500Error(errors.New("boom")).Message)
		require.Equal(t, "boom", throwModifyRegistry400Error(errors.New("boom")).Message)
		require.Equal(t, "boom", throwCreateRegistry400Error(errors.New("boom")).Message)
		require.Equal(t, "boom", throwGetRegistry500Error(errors.New("boom")).Message)
		require.Equal(t, "boom", throwDeleteRegistry500Error(errors.New("boom")).Message)
	})
}
