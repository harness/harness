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

package keywordsearch

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// fakePublicAccess is a test double for publicaccess.Service. It answers Get based on a
// fixed set of public resource paths and records the paths it was queried with. Set / Delete /
// IsPublicAccessSupported are unused by the filter under test.
type fakePublicAccess struct {
	publicPaths map[string]bool
	err         error

	// queriedPaths captures the resource paths passed to Get, in call order.
	queriedPaths []string
}

func (f *fakePublicAccess) Get(
	_ context.Context, _ enum.PublicResourceType, resourcePath string,
) (bool, error) {
	f.queriedPaths = append(f.queriedPaths, resourcePath)
	if f.err != nil {
		return false, f.err
	}
	return f.publicPaths[resourcePath], nil
}

func (f *fakePublicAccess) Set(
	_ context.Context, _ enum.PublicResourceType, _ string, _ bool,
) error {
	return nil
}

func (f *fakePublicAccess) Delete(
	_ context.Context, _ enum.PublicResourceType, _ string,
) error {
	return nil
}

func (f *fakePublicAccess) IsPublicAccessSupported(
	_ context.Context, _ enum.PublicResourceType, _ string,
) (bool, error) {
	return true, nil
}

// fakeAuthorizer is a test double that answers CheckMany based on a fixed map
// keyed by SpacePath. Check / CheckAll are unused by the filter under test.
type fakeAuthorizer struct {
	allowedSpacePaths map[string]bool
	err               error

	// lastChecks captures the checks passed to the most recent CheckMany call.
	lastChecks []types.PermissionCheck

	// returnLen lets a test force a length mismatch between checks and results.
	returnLen int
}

func (f *fakeAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	return true, nil
}

func (f *fakeAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

func (f *fakeAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, permissionChecks ...types.PermissionCheck,
) ([]bool, error) {
	f.lastChecks = append([]types.PermissionCheck(nil), permissionChecks...)
	if f.err != nil {
		return nil, f.err
	}
	n := len(permissionChecks)
	if f.returnLen > 0 {
		n = f.returnLen
	}
	out := make([]bool, n)
	for i := 0; i < len(permissionChecks) && i < n; i++ {
		out[i] = f.allowedSpacePaths[permissionChecks[i].Scope.SpacePath]
	}
	return out, nil
}

func sortedInt64s(in []int64) []int64 {
	out := append([]int64(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestMembershipViewAccessFilter_EmptyRepoMap(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, map[int64]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got: %v", got)
	}
	if len(authorizer.lastChecks) != 0 {
		t.Fatalf("expected no permission checks, got: %d", len(authorizer.lastChecks))
	}
}

func TestMembershipViewAccessFilter_DropsReposAtRoot(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	// "root-repo" has no parent path; paths.Parent returns "".
	repoMap := map[int64]string{
		1: "root-repo",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got: %v", got)
	}
	if len(authorizer.lastChecks) != 0 {
		t.Fatalf("expected no permission checks issued for root repos, got: %d", len(authorizer.lastChecks))
	}
}

func TestMembershipViewAccessFilter_SingleSpaceAllRepos(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{"space": true}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	repoMap := map[int64]string{
		1: "space/repo-a",
		2: "space/repo-b",
		3: "space/repo-c",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{1, 2, 3}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected %v, got %v", want, gotSorted)
	}

	// Verify only ONE permission check was issued (space-level dedup).
	if len(authorizer.lastChecks) != 1 {
		t.Fatalf("expected 1 permission check (deduped by space), got: %d", len(authorizer.lastChecks))
	}
	if authorizer.lastChecks[0].Scope.SpacePath != "space" {
		t.Fatalf("unexpected scope space path: %q", authorizer.lastChecks[0].Scope.SpacePath)
	}
	if authorizer.lastChecks[0].Resource.Type != enum.ResourceTypeRepo {
		t.Fatalf("unexpected resource type: %v", authorizer.lastChecks[0].Resource.Type)
	}
	if authorizer.lastChecks[0].Permission != enum.PermissionRepoView {
		t.Fatalf("unexpected permission: %v", authorizer.lastChecks[0].Permission)
	}
}

func TestMembershipViewAccessFilter_SingleSpaceDenied(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	repoMap := map[int64]string{
		1: "space/repo-a",
		2: "space/repo-b",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result when access denied, got: %v", got)
	}
	if len(authorizer.lastChecks) != 1 {
		t.Fatalf("expected 1 permission check, got: %d", len(authorizer.lastChecks))
	}
}

func TestMembershipViewAccessFilter_MultipleSpacesMixedAccess(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{
		allowedSpacePaths: map[string]bool{
			"space-a": true,
			"space-c": true,
			// "space-b" absent → denied
		},
	}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	repoMap := map[int64]string{
		1: "space-a/repo-1",
		2: "space-a/repo-2",
		3: "space-b/repo-3",
		4: "space-b/repo-4",
		5: "space-c/repo-5",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{1, 2, 5}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected %v, got %v", want, gotSorted)
	}

	if len(authorizer.lastChecks) != 3 {
		t.Fatalf("expected 3 permission checks (one per distinct space), got: %d", len(authorizer.lastChecks))
	}
	seenSpaces := map[string]bool{}
	for _, c := range authorizer.lastChecks {
		seenSpaces[c.Scope.SpacePath] = true
	}
	for _, s := range []string{"space-a", "space-b", "space-c"} {
		if !seenSpaces[s] {
			t.Fatalf("expected a permission check for space %q, got checks for: %v", s, seenSpaces)
		}
	}
}

func TestMembershipViewAccessFilter_NestedSpacesAreDistinct(t *testing.T) {
	t.Parallel()

	// space-a/nested and space-a are different scopes; each needs its own check.
	authorizer := &fakeAuthorizer{
		allowedSpacePaths: map[string]bool{
			"space-a":        true,
			"space-a/nested": false,
		},
	}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	repoMap := map[int64]string{
		1: "space-a/repo-1",
		2: "space-a/nested/repo-2",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{1}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected %v, got %v", want, gotSorted)
	}
	if len(authorizer.lastChecks) != 2 {
		t.Fatalf("expected 2 permission checks (one per distinct space), got: %d", len(authorizer.lastChecks))
	}
}

func TestMembershipViewAccessFilter_AuthorizerError(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	authorizer := &fakeAuthorizer{err: boom}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	_, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, map[int64]string{
		1: "space/repo-a",
	})
	if err == nil {
		t.Fatal("expected error from authorizer")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped authorizer error, got: %v", err)
	}
}

func TestMembershipViewAccessFilter_LengthMismatch(t *testing.T) {
	t.Parallel()

	authorizer := &fakeAuthorizer{
		allowedSpacePaths: map[string]bool{"space": true},
		returnLen:         0, // will be overridden below; keep 0 as sentinel
	}
	// Force a mismatch: return fewer results than checks.
	authorizer.returnLen = 1
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: &fakePublicAccess{}}

	_, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, map[int64]string{
		1: "space-a/repo-1",
		2: "space-b/repo-2",
	})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if got := err.Error(); !strings.Contains(got, "mismatch between number of checks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMembershipViewAccessFilter_PublicRepoInDeniedSpace(t *testing.T) {
	t.Parallel()

	// Membership access to "space" is denied, but repo-b is public and must still be included.
	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{}}
	publicAccess := &fakePublicAccess{publicPaths: map[string]bool{
		"space/repo-b": true,
	}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: publicAccess}

	repoMap := map[int64]string{
		1: "space/repo-a",
		2: "space/repo-b",
		3: "space/repo-c",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{2}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected only the public repo %v, got %v", want, gotSorted)
	}

	// Every repo in the denied space should have been probed for public access.
	wantProbed := sortedStrings([]string{"space/repo-a", "space/repo-b", "space/repo-c"})
	if !equalStrings(sortedStrings(publicAccess.queriedPaths), wantProbed) {
		t.Fatalf("expected public access probes for %v, got %v", wantProbed, publicAccess.queriedPaths)
	}
}

func TestMembershipViewAccessFilter_AllowedSpaceSkipsPublicCheck(t *testing.T) {
	t.Parallel()

	// When membership access to the space is granted, no per-repo public access lookup is needed.
	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{"space": true}}
	publicAccess := &fakePublicAccess{publicPaths: map[string]bool{
		"space/repo-a": true,
	}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: publicAccess}

	repoMap := map[int64]string{
		1: "space/repo-a",
		2: "space/repo-b",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{1, 2}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected all repos %v, got %v", want, gotSorted)
	}
	if len(publicAccess.queriedPaths) != 0 {
		t.Fatalf("expected no public access lookups for an allowed space, got: %v", publicAccess.queriedPaths)
	}
}

func TestMembershipViewAccessFilter_MixedMembershipAndPublic(t *testing.T) {
	t.Parallel()

	// space-a is granted via membership (all repos included, no public probe).
	// space-b is denied, but one of its repos is public.
	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{"space-a": true}}
	publicAccess := &fakePublicAccess{publicPaths: map[string]bool{
		"space-b/repo-4": true,
	}}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: publicAccess}

	repoMap := map[int64]string{
		1: "space-a/repo-1",
		2: "space-a/repo-2",
		3: "space-b/repo-3",
		4: "space-b/repo-4",
	}

	got, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, repoMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSorted := sortedInt64s(got)
	want := []int64{1, 2, 4}
	if !equalInt64s(gotSorted, want) {
		t.Fatalf("expected %v, got %v", want, gotSorted)
	}

	// Only the repos in the denied space-b should have been probed for public access.
	wantProbed := sortedStrings([]string{"space-b/repo-3", "space-b/repo-4"})
	if !equalStrings(sortedStrings(publicAccess.queriedPaths), wantProbed) {
		t.Fatalf("expected public access probes for %v, got %v", wantProbed, publicAccess.queriedPaths)
	}
}

func TestMembershipViewAccessFilter_PublicAccessError(t *testing.T) {
	t.Parallel()

	// The space is denied, forcing a public access lookup that errors out.
	boom := errors.New("public access down")
	authorizer := &fakeAuthorizer{allowedSpacePaths: map[string]bool{}}
	publicAccess := &fakePublicAccess{err: boom}
	filter := membershipViewAccessFilter{authorizer: authorizer, publicAccess: publicAccess}

	_, err := filter.ViewAccessFilter(context.Background(), &auth.Session{}, map[int64]string{
		1: "space/repo-a",
	})
	if err == nil {
		t.Fatal("expected error from public access lookup")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped public access error, got: %v", err)
	}
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
