package githook

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestCheckPushProtection_DoesNotBypassForPlainAPIContent(t *testing.T) {
	t.Parallel()

	manager := protection.NewManager(nil)
	if err := manager.Register(protection.TypePush, func() protection.Definition { return &protection.Push{} }); err != nil {
		t.Fatalf("register push protection: %v", err)
	}

	definition, err := protection.ToJSON(&protection.Push{
		Bypass: protection.DefBypass{
			UserIDs: []int64{101},
		},
		Push: protection.DefPush{
			FileSizeLimit: 1,
		},
	})
	if err != nil {
		t.Fatalf("marshal rule definition: %v", err)
	}

	repoTarget, err := protection.ToJSON(&protection.RepoTarget{})
	if err != nil {
		t.Fatalf("marshal repo target: %v", err)
	}

	controller := &Controller{
		protectionManager: manager,
		settings:          settings.NewService(newInMemorySettingsStore(), refcache.SpaceFinder{}),
		userGroupService:  usergroup.NewService(),
	}

	repo := &types.RepositoryCore{
		ID:         1,
		Identifier: "repo",
		GitUID:     "repo",
	}
	principal := &types.Principal{
		ID:    101,
		Email: "dev@example.com",
	}
	in := types.GithookPreReceiveInput{
		GithookInputBase: types.GithookInputBase{
			RepoID:        repo.ID,
			PrincipalID:   principal.ID,
			OperationType: enum.GitOpTypeAPIContent,
		},
		PreReceiveInput: hook.PreReceiveInput{
			RefUpdates: []hook.ReferenceUpdate{{
				Ref: "refs/heads/main",
				Old: sha.Must("1111111111111111111111111111111111111111"),
				New: sha.Must("2222222222222222222222222222222222222222"),
			}},
		},
	}
	rules := []types.RuleInfoInternal{{
		RuleInfo: types.RuleInfo{
			ID:         1,
			Identifier: "push-protection",
			Type:       protection.TypePush,
			State:      enum.RuleStateActive,
		},
		RepoTarget: repoTarget,
		Definition: definition,
	}}

	violations, _, err := controller.checkPushProtection(
		context.Background(),
		&fakeRestrictedGit{
			processPreReceiveObjectsOutput: git.ProcessPreReceiveObjectsOutput{
				FindOversizeFilesOutput: &git.FindOversizeFilesOutput{
					FileInfosPerLimit: map[int64][]git.FileInfo{
						1: {{SHA: sha.Must("3333333333333333333333333333333333333333"), Size: 2}},
					},
					TotalsPerLimit: map[int64]int64{1: 1},
				},
			},
		},
		repo,
		principal,
		false,
		changedRefs{branches: changes{updated: []string{"main"}}},
		rules,
		in,
		&hook.Output{},
	)
	if err != nil {
		t.Fatalf("check push protection: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 rule violation, got %d", len(violations))
	}
	t.Logf("plain api content violation: bypassable=%v bypassed=%v details=%+v", violations[0].Bypassable, violations[0].Bypassed, violations[0])

	if !violations[0].Bypassable {
		t.Fatalf("expected violation to be bypassable")
	}
	if violations[0].Bypassed {
		t.Fatalf("expected plain API content write not to bypass push protection")
	}
}

func TestCheckPushProtection_DoesBypassForAPIContentBypassRules(t *testing.T) {
	t.Parallel()

	manager := protection.NewManager(nil)
	if err := manager.Register(protection.TypePush, func() protection.Definition { return &protection.Push{} }); err != nil {
		t.Fatalf("register push protection: %v", err)
	}

	definition, err := protection.ToJSON(&protection.Push{
		Bypass: protection.DefBypass{
			UserIDs: []int64{101},
		},
		Push: protection.DefPush{
			FileSizeLimit: 1,
		},
	})
	if err != nil {
		t.Fatalf("marshal rule definition: %v", err)
	}

	repoTarget, err := protection.ToJSON(&protection.RepoTarget{})
	if err != nil {
		t.Fatalf("marshal repo target: %v", err)
	}

	controller := &Controller{
		protectionManager: manager,
		settings:          settings.NewService(newInMemorySettingsStore(), refcache.SpaceFinder{}),
		userGroupService:  usergroup.NewService(),
	}

	repo := &types.RepositoryCore{
		ID:         1,
		Identifier: "repo",
		GitUID:     "repo",
	}
	principal := &types.Principal{
		ID:    101,
		Email: "dev@example.com",
	}
	in := types.GithookPreReceiveInput{
		GithookInputBase: types.GithookInputBase{
			RepoID:        repo.ID,
			PrincipalID:   principal.ID,
			OperationType: enum.GitOpTypeAPIContentBypassRules,
		},
		PreReceiveInput: hook.PreReceiveInput{
			RefUpdates: []hook.ReferenceUpdate{{
				Ref: "refs/heads/main",
				Old: sha.Must("1111111111111111111111111111111111111111"),
				New: sha.Must("2222222222222222222222222222222222222222"),
			}},
		},
	}
	rules := []types.RuleInfoInternal{{
		RuleInfo: types.RuleInfo{
			ID:         1,
			Identifier: "push-protection",
			Type:       protection.TypePush,
			State:      enum.RuleStateActive,
		},
		RepoTarget: repoTarget,
		Definition: definition,
	}}

	violations, _, err := controller.checkPushProtection(
		context.Background(),
		&fakeRestrictedGit{
			processPreReceiveObjectsOutput: git.ProcessPreReceiveObjectsOutput{
				FindOversizeFilesOutput: &git.FindOversizeFilesOutput{
					FileInfosPerLimit: map[int64][]git.FileInfo{
						1: {{SHA: sha.Must("3333333333333333333333333333333333333333"), Size: 2}},
					},
					TotalsPerLimit: map[int64]int64{1: 1},
				},
			},
		},
		repo,
		principal,
		false,
		changedRefs{branches: changes{updated: []string{"main"}}},
		rules,
		in,
		&hook.Output{},
	)
	if err != nil {
		t.Fatalf("check push protection: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 rule violation, got %d", len(violations))
	}
	t.Logf("bypass operation violation: bypassable=%v bypassed=%v details=%+v", violations[0].Bypassable, violations[0].Bypassed, violations[0])
	if !violations[0].Bypassed {
		t.Fatalf("expected API content bypass operation to bypass push protection")
	}
}

type fakeRestrictedGit struct {
	processPreReceiveObjectsOutput git.ProcessPreReceiveObjectsOutput
}

func (f *fakeRestrictedGit) IsAncestor(context.Context, git.IsAncestorParams) (git.IsAncestorOutput, error) {
	return git.IsAncestorOutput{}, nil
}

func (f *fakeRestrictedGit) ScanSecrets(context.Context, *git.ScanSecretsParams) (*git.ScanSecretsOutput, error) {
	return &git.ScanSecretsOutput{}, nil
}

func (f *fakeRestrictedGit) GetBranch(context.Context, *git.GetBranchParams) (*git.GetBranchOutput, error) {
	return &git.GetBranchOutput{}, nil
}

func (f *fakeRestrictedGit) Diff(context.Context, *git.DiffParams, ...api.FileDiffRequest) (<-chan *git.FileDiff, <-chan error) {
	diffCh := make(chan *git.FileDiff)
	errCh := make(chan error)
	close(diffCh)
	close(errCh)
	return diffCh, errCh
}

func (f *fakeRestrictedGit) GetBlob(context.Context, *git.GetBlobParams) (*git.GetBlobOutput, error) {
	return &git.GetBlobOutput{}, nil
}

func (f *fakeRestrictedGit) ProcessPreReceiveObjects(
	context.Context,
	git.ProcessPreReceiveObjectsParams,
) (git.ProcessPreReceiveObjectsOutput, error) {
	return f.processPreReceiveObjectsOutput, nil
}

func (f *fakeRestrictedGit) MergeBase(context.Context, git.MergeBaseParams) (git.MergeBaseOutput, error) {
	return git.MergeBaseOutput{}, nil
}

type inMemorySettingsStore struct {
	values map[string]json.RawMessage
}

func newInMemorySettingsStore() *inMemorySettingsStore {
	return &inMemorySettingsStore{values: map[string]json.RawMessage{}}
}

func (s *inMemorySettingsStore) Find(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) (json.RawMessage, error) {
	value, ok := s.values[s.makeKey(scope, scopeID, key)]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}

	return value, nil
}

func (s *inMemorySettingsStore) FindMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(keys))
	for _, key := range keys {
		value, ok := s.values[s.makeKey(scope, scopeID, key)]
		if ok {
			out[key] = value
		}
	}

	return out, nil
}

func (s *inMemorySettingsStore) Upsert(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
	value json.RawMessage,
) error {
	s.values[s.makeKey(scope, scopeID, key)] = value
	return nil
}

func (s *inMemorySettingsStore) Delete(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) error {
	delete(s.values, s.makeKey(scope, scopeID, key))
	return nil
}

func (s *inMemorySettingsStore) DeleteMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) error {
	for _, key := range keys {
		delete(s.values, s.makeKey(scope, scopeID, key))
	}
	return nil
}

func (s *inMemorySettingsStore) makeKey(scope enum.SettingsScope, scopeID int64, key string) string {
	return fmt.Sprintf("%s:%d:%s", scope, scopeID, key)
}

var _ appstore.SettingsStore = (*inMemorySettingsStore)(nil)
