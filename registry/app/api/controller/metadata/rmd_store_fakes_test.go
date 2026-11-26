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

	"github.com/harness/gitness/app/services/publicaccess"
	gstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// rmdPublicAccess is a public access service fake.
type rmdPublicAccess struct {
	publicaccess.Service
	isPublic     bool
	getErr       error
	supported    bool
	supportedErr error
	setErr       error
	deleteErr    error
	// publicRefs overrides isPublic for specific resource paths when set.
	publicRefs map[string]bool
	setCalls   *[]string
}

func (p rmdPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	if p.getErr != nil {
		return false, p.getErr
	}
	if v, ok := p.publicRefs[path]; ok {
		return v, nil
	}
	return p.isPublic, nil
}

func (p rmdPublicAccess) IsPublicAccessSupported(
	_ context.Context, _ enum.PublicResourceType, _ string,
) (bool, error) {
	return p.supported, p.supportedErr
}

func (p rmdPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, path string, _ bool) error {
	if p.setCalls != nil {
		*p.setCalls = append(*p.setCalls, path)
	}
	return p.setErr
}

func (p rmdPublicAccess) Delete(_ context.Context, _ enum.PublicResourceType, _ string) error {
	return p.deleteErr
}

// rmdSpaceStore is a space store fake.
type rmdSpaceStore struct {
	gstore.SpaceStore
	ancestors   []int64
	err         error
	descendants []int64
	descErr     error
}

func (s rmdSpaceStore) GetDescendantsIDs(_ context.Context, _ int64) ([]int64, error) {
	return s.descendants, s.descErr
}

func (s rmdSpaceStore) GetAncestorIDs(_ context.Context, _ int64) ([]int64, error) {
	return s.ancestors, s.err
}

// rmdRegRepo is a registry repository fake.
type rmdRegRepo struct {
	store.RegistryRepository
	byID           map[int64]*types.Registry
	getErr         error
	byParentName   *types.Registry
	byParentNameE  error
	all            *[]store.RegistryMetadata
	allErr         error
	count          int64
	countErr       error
	createID       int64
	createErr      error
	updateErr      error
	deleteErr      error
	upstreamKeys   []string
	byIDIn         *[]types.Registry
	byIDInErr      error
	usedByIDs      []int64
	usedByErr      error
	deleteCalls    int
	createdEntries []*types.Registry
}

func (r *rmdRegRepo) Get(_ context.Context, id int64, _ ...types.QueryOption) (*types.Registry, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if reg, ok := r.byID[id]; ok {
		return reg, nil
	}
	return &types.Registry{ID: id, UUID: "uuid"}, nil
}

func (r *rmdRegRepo) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Registry, error) {
	return r.byParentName, r.byParentNameE
}

func (r *rmdRegRepo) GetAll(
	_ context.Context, _ []int64, _ []string, _ string, _ string, _ int, _ int, _ string, _ string,
) (*[]store.RegistryMetadata, error) {
	return r.all, r.allErr
}

func (r *rmdRegRepo) CountAll(
	_ context.Context, _ []int64, _ []string, _ string, _ string,
) (int64, error) {
	return r.count, r.countErr
}

func (r *rmdRegRepo) Create(_ context.Context, reg *types.Registry) (int64, error) {
	r.createdEntries = append(r.createdEntries, reg)
	return r.createID, r.createErr
}

func (r *rmdRegRepo) Update(_ context.Context, _ *types.Registry) error {
	return r.updateErr
}

func (r *rmdRegRepo) Delete(_ context.Context, _ int64, _ string) error {
	r.deleteCalls++
	return r.deleteErr
}

func (r *rmdRegRepo) FetchUpstreamProxyKeys(
	_ context.Context, _ []int64, _ ...types.QueryOption,
) ([]string, error) {
	return r.upstreamKeys, nil
}

func (r *rmdRegRepo) GetByIDIn(
	_ context.Context, _ []int64, _ ...types.QueryOption,
) (*[]types.Registry, error) {
	return r.byIDIn, r.byIDInErr
}

func (r *rmdRegRepo) FetchRegistriesIDByUpstreamProxyID(
	_ context.Context, _ string, _ int64, _ ...types.QueryOption,
) ([]int64, error) {
	return r.usedByIDs, r.usedByErr
}

// rmdTx runs the transaction function inline.
type rmdTx struct {
	err error
}

func (t rmdTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

// rmdAudit records audit entries.
type rmdAudit struct {
	err     error
	entries *[]string
}

func (a rmdAudit) Log(
	_ context.Context, _ coretypes.Principal, resource audit.Resource, action audit.Action, _ string,
	_ ...audit.Option,
) error {
	if a.entries != nil {
		*a.entries = append(*a.entries, string(action)+":"+resource.Identifier)
	}
	return a.err
}

// rmdRegFinder is a registry finder fake.
type rmdRegFinder struct {
	refcache.RegistryFinder
	updateErr error
	deleteErr error
	found     *types.Registry
	findErr   error
}

func (f rmdRegFinder) Update(_ context.Context, _ *types.Registry) error { return f.updateErr }

func (f rmdRegFinder) Delete(_ context.Context, _ int64, _ string) error { return f.deleteErr }

func (f rmdRegFinder) FindByRootParentID(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Registry, error) {
	return f.found, f.findErr
}

// rmdUpstreamRepo is an upstream proxy config repository fake.
type rmdUpstreamRepo struct {
	store.UpstreamProxyConfigRepository
	byIdent    *types.UpstreamProxy
	byIdentErr error
	byID       *types.UpstreamProxy
	byIDErr    error
	createErr  error
	updateErr  error
}

func (r rmdUpstreamRepo) GetByRegistryIdentifier(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.UpstreamProxy, error) {
	return r.byIdent, r.byIdentErr
}

func (r rmdUpstreamRepo) Get(_ context.Context, _ int64, _ ...types.QueryOption) (*types.UpstreamProxy, error) {
	return r.byID, r.byIDErr
}

func (r rmdUpstreamRepo) Create(_ context.Context, _ *types.UpstreamProxyConfig) (int64, error) {
	return 1, r.createErr
}

func (r rmdUpstreamRepo) Update(_ context.Context, _ *types.UpstreamProxyConfig) error {
	return r.updateErr
}

// rmdCleanupRepo is a cleanup policy repository fake.
type rmdCleanupRepo struct {
	store.CleanupPolicyRepository
	policies  *[]types.CleanupPolicy
	getErr    error
	idsErr    error
	modifyErr error
}

func (r rmdCleanupRepo) GetByRegistryID(_ context.Context, _ int64) (*[]types.CleanupPolicy, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.policies == nil {
		return &[]types.CleanupPolicy{}, nil
	}
	return r.policies, nil
}

func (r rmdCleanupRepo) GetIDsByRegistryID(_ context.Context, _ int64) ([]int64, error) {
	return nil, r.idsErr
}

func (r rmdCleanupRepo) ModifyCleanupPolicies(
	_ context.Context, _ *[]types.CleanupPolicy, _ []int64,
) error {
	return r.modifyErr
}

// rmdNewCtrl builds a controller where every dependency is a permissive fake.
// Tests override individual fields through the opts functions.
func rmdNewCtrl(opts ...func(*APIController)) *APIController {
	c := &APIController{
		RegistryMetadataHelper: rmdHelper{info: rmdBaseInfo()},
		SpaceFinder:            rmdSpaceFinder{space: rmdSpaceCore()},
		Authorizer:             rmdAuthorizer{allow: true},
		RegistryRepository:     &rmdRegRepo{},
		UpstreamProxyStore:     rmdUpstreamRepo{},
		CleanupPolicyStore:     rmdCleanupRepo{},
		PackageWrapper:         rmdPkgWrapper{},
		PublicAccess:           rmdPublicAccess{supported: true},
		SpaceStore:             rmdSpaceStore{},
		RegFinder:              rmdRegFinder{},
		AuditService:           rmdAudit{},
		URLProvider:            rmdURLProvider{},
		tx:                     rmdTx{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

var rmdPrincipal = coretypes.Principal{ID: 7, Type: enum.PrincipalTypeUser, Email: "dev@example.com"}

type rmdSpaceResult struct {
	space *coretypes.SpaceCore
	err   error
}

// rmdSeqSpaceFinder returns the configured results in order, one per lookup.
type rmdSeqSpaceFinder struct {
	results []rmdSpaceResult
	idx     int
}

func (f *rmdSeqSpaceFinder) next() (*coretypes.SpaceCore, error) {
	if f.idx >= len(f.results) {
		return f.results[len(f.results)-1].space, f.results[len(f.results)-1].err
	}
	r := f.results[f.idx]
	f.idx++
	return r.space, r.err
}

func (f *rmdSeqSpaceFinder) FindByID(_ context.Context, _ int64) (*coretypes.SpaceCore, error) {
	return f.next()
}

func (f *rmdSeqSpaceFinder) FindByRef(_ context.Context, _ string) (*coretypes.SpaceCore, error) {
	return f.next()
}

// rmdWebhookRepo is a webhooks repository fake.
type rmdWebhookRepo struct {
	store.WebhooksRepository
	byIdent    *coretypes.WebhookCore
	byIdentErr error
	// secondGetErr is returned from the second GetByRegistryAndIdentifier call.
	secondGetErr error
	calls        *int
	list         []*coretypes.WebhookCore
	listErr      error
	count        int64
	countErr     error
	createErr    error
	updateErr    error
	deleteErr    error
	updated      **coretypes.WebhookCore
}

func (r rmdWebhookRepo) GetByRegistryAndIdentifier(
	_ context.Context, _ int64, _ string,
) (*coretypes.WebhookCore, error) {
	if r.calls != nil {
		*r.calls++
		if *r.calls > 1 && r.secondGetErr != nil {
			return nil, r.secondGetErr
		}
	}
	return r.byIdent, r.byIdentErr
}

func (r rmdWebhookRepo) ListByRegistry(
	_ context.Context, _ string, _ string, _ int, _ int, _ string, _ int64,
) ([]*coretypes.WebhookCore, error) {
	return r.list, r.listErr
}

func (r rmdWebhookRepo) CountAllByRegistry(_ context.Context, _ int64, _ string) (int64, error) {
	return r.count, r.countErr
}

func (r rmdWebhookRepo) Create(_ context.Context, _ *coretypes.WebhookCore) error { return r.createErr }

func (r rmdWebhookRepo) Update(_ context.Context, w *coretypes.WebhookCore) error {
	if r.updated != nil {
		*r.updated = w
	}
	return r.updateErr
}

func (r rmdWebhookRepo) DeleteByRegistryAndIdentifier(_ context.Context, _ int64, _ string) error {
	return r.deleteErr
}
