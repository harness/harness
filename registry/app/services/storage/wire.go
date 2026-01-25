package storage

import "github.com/harness/gitness/registry/app/storage"

func ProvideBlobCreationDBHook() storage.BlobCreationDBHook {
	return NewBlobCreationDBHook()
}
