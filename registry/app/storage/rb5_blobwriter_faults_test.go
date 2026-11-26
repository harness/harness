//  Copyright 2023 Harness, Inc.
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

package storage

import (
	"context"
	"io"
	"path"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// rb5W is the surface shared by blobWriter and globalBlobWriter.
type rb5W interface {
	BlobWriter
	Reader() (io.ReadCloser, error)
	validateBlob(ctx context.Context, desc manifest.Descriptor) (manifest.Descriptor, error)
	moveBlob(ctx context.Context, pathPrefix string, desc manifest.Descriptor) error
	removeResources(ctx context.Context) error
	resumeDigest(ctx context.Context) error
	storeHashState(ctx context.Context) error
	getStoredHashStates(ctx context.Context) ([]hashStateEntry, error)
}

type rb5Kit struct {
	name      string
	fd        *rb5Driver
	prefix    string
	store     rb5BlobReader
	create    func(multipart bool) rb5W
	resume    func(id string, multipart bool) rb5W
	statePath func(id string, offset int64) string
	noDigest  func(w rb5W)
}

func rb5OciKit(t *testing.T) *rb5Kit {
	t.Helper()
	ctx := context.Background()
	fd := rb5NewDriver(t)
	bs := rb5OciStore(fd)
	bs.resumableDigestEnabled = true
	wrap := func(w BlobWriter, err error, multipart bool) rb5W {
		require.NoError(t, err)
		bw, ok := w.(*blobWriter)
		require.True(t, ok)
		bw.isMultiPart = multipart
		return bw
	}
	return &rb5Kit{
		name: "oci", fd: fd, prefix: rb5Prefix, store: bs,
		create: func(multipart bool) rb5W {
			w, err := bs.Create(ctx)
			return wrap(w, err, multipart)
		},
		resume: func(id string, multipart bool) rb5W {
			w, err := bs.Resume(ctx, id)
			return wrap(w, err, multipart)
		},
		statePath: func(id string, offset int64) string {
			p, err := pathFor(uploadHashStatePathSpec{
				path: rb5Prefix, repoName: rb5Repo, id: id, alg: digest.SHA256, offset: offset,
			})
			require.NoError(t, err)
			return p
		},
		noDigest: func(w rb5W) {
			if bw, ok := w.(*blobWriter); ok {
				bw.resumableDigestEnabled = false
			}
		},
	}
}

func rb5GlobalKit(t *testing.T) *rb5Kit {
	t.Helper()
	ctx := context.Background()
	fd := rb5NewDriver(t)
	bs := rb5GlobalStore(fd)
	wrap := func(w BlobWriter, err error, multipart bool) rb5W {
		require.NoError(t, err)
		bw, ok := w.(*globalBlobWriter)
		require.True(t, ok)
		bw.isMultiPart = multipart
		return bw
	}
	return &rb5Kit{
		name: "global", fd: fd, prefix: "", store: bs,
		create: func(multipart bool) rb5W {
			w, err := bs.Create(ctx)
			return wrap(w, err, multipart)
		},
		resume: func(id string, multipart bool) rb5W {
			w, err := bs.Resume(ctx, id)
			return wrap(w, err, multipart)
		},
		statePath: func(id string, offset int64) string {
			p, err := pathFor(globalUploadHashStatePathSpec{id: id, alg: digest.SHA256, offset: offset})
			require.NoError(t, err)
			return p
		},
		noDigest: func(w rb5W) {
			if bw, ok := w.(*globalBlobWriter); ok {
				bw.resumableDigestEnabled = false
			}
		},
	}
}

func rb5Kits(t *testing.T, fn func(t *testing.T, k *rb5Kit)) {
	t.Helper()
	for name, newKit := range map[string]func(*testing.T) *rb5Kit{"oci": rb5OciKit, "global": rb5GlobalKit} {
		t.Run(name, func(t *testing.T) { fn(t, newKit(t)) })
	}
}

func rb5Desc(payload string) manifest.Descriptor {
	return manifest.Descriptor{Digest: digest.FromString(payload), Size: int64(len(payload))}
}

// rb5Written creates a non-multipart upload holding the payload on disk.
func rb5Written(t *testing.T, k *rb5Kit, payload string) rb5W {
	t.Helper()
	w := k.create(false)
	_, err := w.Write([]byte(payload))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return w
}

func TestRb5WriterCommit(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := k.create(true)
		_, err := w.Write([]byte(rb5Payload))
		require.NoError(t, err)
		require.Equal(t, int64(len(rb5Payload)), w.Size())
		require.NotEmpty(t, w.ID())

		desc, err := w.Commit(ctx, k.prefix, rb5Desc(rb5Payload))
		require.NoError(t, err)
		require.Equal(t, rb5Dgst(), desc.Digest)
		require.Equal(t, "application/octet-stream", desc.MediaType)

		_, err = k.store.Stat(ctx, k.prefix, rb5Dgst())
		require.NoError(t, err)

		// Closing after a commit is rejected for multipart writers.
		require.ErrorContains(t, w.Close(), "close after commit")
	})
}

func TestRb5WriterCommitFailures(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func(k *rb5Kit)
		desc  manifest.Descriptor
	}{
		{
			name:  "file writer commit",
			setup: func(k *rb5Kit) { k.fd.writerFault.commit = errRb5Boom },
			desc:  rb5Desc(rb5Payload),
		},
		{name: "empty digest", setup: func(*rb5Kit) {}, desc: manifest.Descriptor{}},
		{
			name:  "digest mismatch",
			setup: func(*rb5Kit) {},
			desc:  manifest.Descriptor{Digest: digest.FromString("something else")},
		},
		{name: "move", setup: func(k *rb5Kit) { k.fd.fail["Move"] = errRb5Boom }, desc: rb5Desc(rb5Payload)},
		{name: "remove resources", setup: func(k *rb5Kit) { k.fd.fail["Delete"] = errRb5Boom }, desc: rb5Desc(rb5Payload)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb5Kits(t, func(t *testing.T, k *rb5Kit) {
				w := k.create(false)
				_, err := w.Write([]byte(rb5Payload))
				require.NoError(t, err)
				tt.setup(k)
				_, err = w.Commit(ctx, k.prefix, tt.desc)
				require.Error(t, err)
			})
		})
	}
}

func TestRb5WriterResumeFlow(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		first := k.create(true)
		_, err := first.Write([]byte("abc"))
		require.NoError(t, err)
		require.NoError(t, first.Close())

		second := k.resume(first.ID(), true)
		_, err = second.Write([]byte("def"))
		require.NoError(t, err)

		desc, err := second.Commit(ctx, k.prefix, rb5Desc("abcdef"))
		require.NoError(t, err)
		require.Equal(t, digest.FromString("abcdef"), desc.Digest)
	})
}

func TestRb5WriterResumeDigestErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func(k *rb5Kit, id string)
		wantErr string
	}{
		{
			name:    "list fails",
			setup:   func(k *rb5Kit, _ string) { k.fd.fail["List"] = errRb5Boom },
			wantErr: "stored hash states",
		},
		{
			name:    "state unreadable",
			setup:   func(k *rb5Kit, _ string) { k.fd.fail["GetContent"] = errRb5Boom },
			wantErr: "boom",
		},
		{
			name: "state corrupt",
			setup: func(k *rb5Kit, id string) {
				require.NoError(t, k.fd.PutContent(ctx, k.statePath(id, 3), []byte("garbage")))
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb5Kits(t, func(t *testing.T, k *rb5Kit) {
				first := k.create(true)
				_, err := first.Write([]byte("abc"))
				require.NoError(t, err)
				require.NoError(t, first.Close())

				second := k.resume(first.ID(), true)
				tt.setup(k, first.ID())
				err = second.resumeDigest(ctx)
				require.Error(t, err)
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
				}
				_, err = second.Write([]byte("d"))
				require.Error(t, err)
			})
		})
	}
}

func TestRb5WriterResumeDigestFallbacks(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		// No stored states: the digest cannot be resumed but writing still works.
		first := rb5Written(t, k, "abc")
		second := k.resume(first.ID(), true)
		require.ErrorIs(t, second.resumeDigest(ctx), errResumableDigestNotAvailable)
		_, err := second.Write([]byte("d"))
		require.NoError(t, err)
		require.NoError(t, second.Close())

		// A stray non numeric entry in the hash state directory is skipped.
		require.NoError(t, k.fd.PutContent(ctx, path.Join(path.Dir(k.statePath(first.ID(), 0)), "notanumber"), []byte("x")))
		states, err := second.getStoredHashStates(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, states)

		// Resumable digests disabled.
		k.noDigest(second)
		require.ErrorIs(t, second.resumeDigest(ctx), errResumableDigestNotAvailable)
		require.ErrorIs(t, second.storeHashState(ctx), errResumableDigestNotAvailable)
	})
}

func TestRb5WriterHashStateStoreFailures(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := k.create(true)
		_, err := w.Write([]byte("abc"))
		require.NoError(t, err)

		k.fd.fail["PutContent"] = errRb5Boom
		require.ErrorIs(t, w.storeHashState(ctx), errRb5Boom)
		require.ErrorIs(t, w.Close(), errRb5Boom)
		delete(k.fd.fail, "PutContent")

		k.fd.fail["List"] = errRb5Boom
		_, err = w.getStoredHashStates(ctx)
		require.ErrorIs(t, err, errRb5Boom)
	})
}

func TestRb5WriterWriteAndCloseFailures(t *testing.T) {
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		for _, multipart := range []bool{false, true} {
			w := k.create(multipart)
			k.fd.writerFault.write = errRb5Boom
			_, err := w.Write([]byte("abc"))
			require.ErrorIs(t, err, errRb5Boom)
			k.fd.writerFault.write = nil

			k.fd.writerFault.closeE = errRb5Boom
			require.ErrorIs(t, w.Close(), errRb5Boom)
			k.fd.writerFault.closeE = nil
		}
	})
}

func TestRb5WriterValidateBlob(t *testing.T) {
	ctx := context.Background()
	emptyDesc := manifest.Descriptor{Digest: digest.FromString("")}

	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := rb5Written(t, k, rb5Payload)

		_, err := w.validateBlob(ctx, manifest.Descriptor{})
		require.ErrorAs(t, err, &BlobInvalidDigestError{})

		k.fd.seq["Stat"] = []error{errRb5Boom}
		_, err = w.validateBlob(ctx, rb5Desc(rb5Payload))
		require.ErrorIs(t, err, errRb5Boom)

		k.fd.dirStat = true
		_, err = w.validateBlob(ctx, rb5Desc(rb5Payload))
		require.ErrorContains(t, err, "unexpected directory")
		k.fd.dirStat = false

		bad := rb5Desc(rb5Payload)
		bad.Size = 99
		_, err = w.validateBlob(ctx, bad)
		require.ErrorIs(t, err, ErrBlobInvalidLength)

		mismatch := manifest.Descriptor{Digest: digest.FromString("other")}
		_, err = w.validateBlob(ctx, mismatch)
		require.ErrorAs(t, err, &BlobInvalidDigestError{})

		desc, err := w.validateBlob(ctx, manifest.Descriptor{Digest: rb5Dgst()})
		require.NoError(t, err)
		require.Equal(t, int64(len(rb5Payload)), desc.Size)

		// The upload file is missing, so the size is treated as zero.
		empty := k.create(false)
		k.fd.seq["Stat"] = []error{rb5NotFound()}
		desc, err = empty.validateBlob(ctx, emptyDesc)
		require.NoError(t, err)
		require.Zero(t, desc.Size)

		// A different digest algorithm forces a full hash and returns the canonical digest.
		sha512Desc := manifest.Descriptor{Digest: digest.SHA512.FromString(rb5Payload)}
		desc, err = w.validateBlob(ctx, sha512Desc)
		require.NoError(t, err)
		require.Equal(t, rb5Dgst(), desc.Digest)

		// Reading the stored data back fails.
		k.fd.fail["Reader"] = errRb5Boom
		_, err = w.validateBlob(ctx, rb5Desc(rb5Payload))
		require.Error(t, err)
	})
}

func TestRb5WriterValidateBlobMultipart(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := k.create(true)
		_, err := w.Write([]byte(rb5Payload))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// The digester has seen all data, so the resumed digest is used directly.
		_, err = w.validateBlob(ctx, rb5Desc(rb5Payload))
		require.NoError(t, err)

		// Another algorithm needs a full hash of the stored data.
		_, err = w.validateBlob(ctx, manifest.Descriptor{Digest: digest.SHA512.FromString(rb5Payload)})
		require.NoError(t, err)

		// Resuming the digest fails with an unexpected error once the offset and written bytes diverge.
		second := k.resume(w.ID(), true)
		k.fd.fail["List"] = errRb5Boom
		_, err = second.validateBlob(ctx, rb5Desc(rb5Payload))
		require.ErrorContains(t, err, "stored hash states")
	})
}

func TestRb5WriterValidateBlobWrittenMatchesSize(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := k.create(true)
		k.noDigest(w)
		_, err := w.Write([]byte(rb5Payload))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// Resumable digests are unavailable but everything was written by this writer.
		desc, err := w.validateBlob(ctx, rb5Desc(rb5Payload))
		require.NoError(t, err)
		require.Equal(t, rb5Dgst(), desc.Digest)
	})
}

func TestRb5WriterMoveBlob(t *testing.T) {
	ctx := context.Background()
	desc := rb5Desc(rb5Payload)
	empty := manifest.Descriptor{Digest: digestSha256Empty}

	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := rb5Written(t, k, rb5Payload)

		require.Error(t, w.moveBlob(ctx, k.prefix, manifest.Descriptor{Digest: rb5BadDigest}))

		k.fd.seq["Stat"] = []error{errRb5Boom}
		require.ErrorIs(t, w.moveBlob(ctx, k.prefix, desc), errRb5Boom)

		k.fd.seq["Stat"] = []error{rb5NotFound(), errRb5Boom}
		require.ErrorIs(t, w.moveBlob(ctx, k.prefix, desc), errRb5Boom)

		k.fd.seq["Stat"] = []error{rb5NotFound(), rb5NotFound()}
		k.fd.fail["PutContent"] = errRb5Boom
		require.ErrorIs(t, w.moveBlob(ctx, k.prefix, empty), errRb5Boom)
		delete(k.fd.fail, "PutContent")

		k.fd.seq["Stat"] = []error{rb5NotFound(), rb5NotFound()}
		require.NoError(t, w.moveBlob(ctx, k.prefix, empty))

		k.fd.seq["Stat"] = []error{rb5NotFound(), rb5NotFound()}
		k.fd.fail["Move"] = errRb5Boom
		require.ErrorIs(t, w.moveBlob(ctx, k.prefix, desc), errRb5Boom)
		delete(k.fd.fail, "Move")

		require.NoError(t, w.moveBlob(ctx, k.prefix, desc))
		// The blob now exists, so a second move is a no-op.
		require.NoError(t, w.moveBlob(ctx, k.prefix, desc))
	})
}

func TestRb5WriterRemoveResourcesAndCancel(t *testing.T) {
	ctx := context.Background()
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := rb5Written(t, k, rb5Payload)

		k.fd.fail["Delete"] = errRb5Boom
		require.ErrorIs(t, w.removeResources(ctx), errRb5Boom)
		k.fd.fail["Delete"] = rb5NotFound()
		require.NoError(t, w.removeResources(ctx))
		delete(k.fd.fail, "Delete")

		c := k.create(false)
		k.fd.writerFault.cancel = errRb5Boom
		require.ErrorIs(t, c.Cancel(ctx), errRb5Boom)
		k.fd.writerFault.cancel = nil

		k.fd.writerFault.closeE = errRb5Boom
		k.fd.fail["Delete"] = errRb5Boom
		require.ErrorIs(t, c.Cancel(ctx), errRb5Boom)
		delete(k.fd.fail, "Delete")
		k.fd.writerFault.closeE = nil

		d := k.create(false)
		require.NoError(t, d.Cancel(ctx))
	})
}

func TestRb5WriterReader(t *testing.T) {
	rb5Kits(t, func(t *testing.T, k *rb5Kit) {
		w := rb5Written(t, k, rb5Payload)

		rc, err := w.Reader()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, rb5Payload, string(data))
		require.NoError(t, rc.Close())

		k.fd.seq["Stat"] = []error{errRb5Boom}
		_, err = w.Reader()
		require.ErrorIs(t, err, errRb5Boom)

		// The upload file shows up after one retry.
		k.fd.seq["Stat"] = []error{rb5NotFound(), errRb5Pass}
		rc, err = w.Reader()
		require.NoError(t, err)
		require.NoError(t, rc.Close())

		k.fd.fail["Reader"] = errRb5Boom
		_, err = w.Reader()
		require.ErrorIs(t, err, errRb5Boom)
	})
}

func TestRb5WriterPlainCommit(t *testing.T) {
	ctx := context.Background()
	sha := "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"

	type plainKit struct {
		name   string
		fd     *rb5Driver
		create func() BlobWriter
	}
	newKits := func(t *testing.T) []plainKit {
		t.Helper()
		gfd := rb5NewDriver(t)
		gs := rb5GlobalStore(gfd)
		bfd := rb5NewDriver(t)
		bs := &genericBlobStore{driver: bfd, rootParentRef: rb5Prefix}
		return []plainKit{
			{name: "global", fd: gfd, create: func() BlobWriter {
				w, err := gs.CreateGeneric(ctx, "root")
				require.NoError(t, err)
				return w
			}},
			{name: "generic", fd: bfd, create: func() BlobWriter {
				w, err := bs.CreateGeneric(ctx, "root")
				require.NoError(t, err)
				return w
			}},
		}
	}

	tests := []struct {
		name    string
		setup   func(fd *rb5Driver)
		wantErr bool
	}{
		{name: "success", setup: func(*rb5Driver) {}},
		{name: "commit fails", setup: func(fd *rb5Driver) { fd.writerFault.commit = errRb5Boom }, wantErr: true},
		{name: "close fails", setup: func(fd *rb5Driver) { fd.writerFault.closeE = errRb5Boom }, wantErr: true},
		{name: "move fails", setup: func(fd *rb5Driver) { fd.fail["Move"] = errRb5Boom }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range newKits(t) {
				w := k.create()
				_, err := w.Write([]byte("abc"))
				require.NoError(t, err, k.name)
				tt.setup(k.fd)
				err = w.PlainCommit(ctx, sha)
				if tt.wantErr {
					require.Error(t, err, k.name)
				} else {
					require.NoError(t, err, k.name)
				}
			}
		})
	}
}
