package blobsql

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/server/blobstore"
	"github.com/russross/meddler"
)

func NewContext(parent context.Context, db meddler.DB) context.Context {
	return blobstore.NewContext(parent, New(db))
}
