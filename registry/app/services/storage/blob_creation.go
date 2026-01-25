package storage

import (
	"context"

	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
)

type blobCreationDBHook struct {
}

func (b *blobCreationDBHook) AfterBlobCreate(
	ctx context.Context,
	rootParentID int64,
	sha1 types.Digest,
	sha256 types.Digest,
	sha512 types.Digest,
	md5 types.Digest,
	size int64,
	bucketID int64,
) error {
	return nil
}

func NewBlobCreationDBHook() storage.BlobCreationDBHook {
	return &blobCreationDBHook{}
}
