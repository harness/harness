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

package lfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errCtlBlob = errors.New("blob failure")

// ctlBlobStore is an in-memory blob.Store.
type ctlBlobStore struct {
	files      map[string][]byte
	uploadErr  error
	moveErr    error
	deleteErr  error
	downloadEr error
}

func (s *ctlBlobStore) Upload(_ context.Context, file io.Reader, filePath string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	s.files[filePath] = data
	return nil
}

func (s *ctlBlobStore) GetSignedURL(context.Context, string, time.Time, ...blob.SignURLOption) (string, error) {
	return "", nil
}

func (s *ctlBlobStore) Download(_ context.Context, filePath string) (io.ReadCloser, error) {
	if s.downloadEr != nil {
		return nil, s.downloadEr
	}
	data, ok := s.files[filePath]
	if !ok {
		return nil, blob.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *ctlBlobStore) Move(_ context.Context, src, dst string) error {
	if s.moveErr != nil {
		return s.moveErr
	}
	data, ok := s.files[src]
	if !ok {
		return blob.ErrNotFound
	}
	s.files[dst] = data
	delete(s.files, src)
	return nil
}

func (s *ctlBlobStore) Delete(_ context.Context, filePath string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if _, ok := s.files[filePath]; !ok {
		return blob.ErrNotFound
	}
	delete(s.files, filePath)
	return nil
}

type ctlURLProvider struct {
	url.Provider
}

func (ctlURLProvider) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

type ctlRemoteAuth struct{ err error }

func (r ctlRemoteAuth) GenerateToken(context.Context, int64, enum.PrincipalType, string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return "jwt-token", nil
}

type ctlEnv struct {
	*ctlBase
	ctrl     *Controller
	blobs    *ctlBlobStore
	lfs      *database.LFSObjectStore
	settings *settings.Service
	repo     *types.Repository
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	settingsSvc := settings.NewService(database.NewSettingsStore(b.db), b.spaceFinder)
	lfsStore := database.NewLFSObjectStore(b.db)
	blobs := &ctlBlobStore{files: map[string][]byte{}}

	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")

	return &ctlEnv{
		ctlBase: b,
		ctrl: NewController(
			ctlAuthorizer{mode: ctlAllow}, b.repoFinder, b.repos, b.principals, lfsStore, blobs,
			ctlRemoteAuth{}, ctlURLProvider{}, settingsSvc,
		),
		blobs:    blobs,
		lfs:      lfsStore,
		settings: settingsSvc,
		repo:     repo,
	}
}

func ctlOID(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestCtlLFSController_UploadDownload(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	content := "hello lfs"
	ptr := Pointer{OId: ctlOID(content), Size: int64(len(content))}

	out, err := e.ctrl.Upload(ctx, s, "root/repo1", ptr, strings.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, "lfs/"+ptr.OId, out.ObjectPath)

	dl, err := e.ctrl.Download(ctx, s, "root/repo1", ptr.OId)
	require.NoError(t, err)
	data, err := io.ReadAll(dl)
	require.NoError(t, err)
	require.NoError(t, dl.Close())
	require.Equal(t, content, string(data))
	require.EqualValues(t, len(content), dl.Size)

	// Existing objects cannot be overwritten.
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", ptr, strings.NewReader(content))
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 409, userErr.Status)

	// Unknown object.
	_, err = e.ctrl.Download(ctx, s, "root/repo1", ctlOID("nope"))
	require.Error(t, err)

	// Object exists in the store but not in the blob store.
	require.NoError(t, e.lfs.Create(ctx, &types.LFSObject{
		OID: "ghost", Size: 1, RepoID: e.repo.ID, CreatedBy: e.admin.ID,
	}))
	_, err = e.ctrl.DownloadNoAuth(ctx, e.repo.ID, "ghost")
	require.ErrorIs(t, err, blob.ErrNotFound)
}

func TestCtlLFSController_UploadErrors(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	content := "payload"
	ptr := Pointer{OId: ctlOID(content), Size: int64(len(content))}

	_, err := e.ctrl.Upload(ctx, s, "root/repo1", ptr, nil)
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 400, userErr.Status)

	// Hash mismatch removes the temporary file.
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", Pointer{OId: ctlOID("other"), Size: ptr.Size}, strings.NewReader(content))
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 400, userErr.Status)
	require.Empty(t, e.blobs.files)

	// Hash mismatch where the cleanup fails as well (not found is tolerated, others are only logged).
	e.blobs.deleteErr = errCtlBlob
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", Pointer{OId: ctlOID("other"), Size: ptr.Size}, strings.NewReader(content))
	require.ErrorAs(t, err, &userErr)
	e.blobs.deleteErr = nil

	e.blobs.uploadErr = errCtlBlob
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", ptr, strings.NewReader(content))
	require.ErrorIs(t, err, errCtlBlob)
	e.blobs.uploadErr = nil

	e.blobs.moveErr = errCtlBlob
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", ptr, strings.NewReader(content))
	require.ErrorIs(t, err, errCtlBlob)
	e.blobs.deleteErr = errCtlBlob
	_, err = e.ctrl.Upload(ctx, s, "root/repo1", ptr, strings.NewReader(content))
	require.ErrorIs(t, err, errCtlBlob)
}

func TestCtlLFSController_Transfer(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	existing := Pointer{OId: "existing", Size: 5}
	require.NoError(t, e.lfs.Create(ctx, &types.LFSObject{
		OID: existing.OId, Size: existing.Size, RepoID: e.repo.ID, CreatedBy: e.admin.ID,
	}))
	missing := Pointer{OId: "missing", Size: 7}

	// Download.
	out, err := e.ctrl.LFSTransfer(ctx, s, "root/repo1", &TransferInput{
		Operation: enum.GitLFSOperationTypeDownload, Objects: []Pointer{existing, missing},
	})
	require.NoError(t, err)
	require.Equal(t, enum.GitLFSTransferTypeBasic, out.Transfer)
	require.Len(t, out.Objects, 2)
	require.Contains(t, out.Objects[0].Actions["download"].Href, "oid=existing")
	require.NotContains(t, out.Objects[0].Actions["download"].Href, "size=")
	require.NotNil(t, out.Objects[1].Error)

	// Upload.
	out, err = e.ctrl.LFSTransfer(ctx, s, "root/repo1", &TransferInput{
		Operation: enum.GitLFSOperationTypeUpload, Objects: []Pointer{existing, missing},
	})
	require.NoError(t, err)
	require.Len(t, out.Objects, 2)
	require.Empty(t, out.Objects[0].Actions)
	require.Contains(t, out.Objects[1].Actions["upload"].Href, "oid=missing&size=7")

	// Unsupported operation.
	_, err = e.ctrl.LFSTransfer(ctx, s, "root/repo1", &TransferInput{Operation: "bogus"})
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 400, userErr.Status)
}

func TestCtlLFSController_Authenticate(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	resp, err := e.ctrl.Authenticate(ctx, s, "root/repo1")
	require.NoError(t, err)
	require.Contains(t, resp.Header["Authorization"], "jwt-token")
	require.Equal(t, "https://git.example.com/root/repo1.git/info/lfs", resp.HRef)
	require.NotZero(t, resp.ExpiresIn)

	_, err = e.ctrl.Authenticate(ctx, s, "root/unknown")
	require.Error(t, err)

	c := *e.ctrl
	c.remoteAuth = ctlRemoteAuth{err: errCtlBlob}
	_, err = c.Authenticate(ctx, s, "root/repo1")
	require.ErrorIs(t, err, errCtlBlob)

	require.NoError(t, e.settings.RepoSet(ctx, e.repo.ID, settings.KeyGitLFSEnabled, false))
	_, err = e.ctrl.Authenticate(ctx, s, "root/repo1")
	require.ErrorIs(t, err, usererror.ErrGitLFSDisabled)
}

func TestCtlLFSController_AccessAndSettings(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	in := &TransferInput{Operation: enum.GitLFSOperationTypeDownload}
	ptr := Pointer{OId: "x", Size: 1}

	call := func(c *Controller, repoRef string) map[string]error {
		errs := map[string]error{}
		_, errs["transfer"] = c.LFSTransfer(ctx, s, repoRef, in)
		_, errs["download"] = c.Download(ctx, s, repoRef, "x")
		_, errs["upload"] = c.Upload(ctx, s, repoRef, ptr, strings.NewReader("x"))
		return errs
	}

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *e.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}
		for name, err := range call(&c, "root/repo1") {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			}
		}
	}

	for name, err := range call(e.ctrl, "") {
		require.Error(t, err, name)
	}
	for name, err := range call(e.ctrl, "root/unknown") {
		require.Error(t, err, name)
	}

	require.NoError(t, e.settings.RepoSet(ctx, e.repo.ID, settings.KeyGitLFSEnabled, false))
	for name, err := range call(e.ctrl, "root/repo1") {
		require.ErrorIs(t, err, usererror.ErrGitLFSDisabled, name)
	}
}

func TestCtlLFSContent(t *testing.T) {
	c := &Content{Data: io.NopCloser(strings.NewReader("abc")), Size: 3}
	buf := make([]byte, 3)
	n, err := c.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.NoError(t, c.Close())
}
