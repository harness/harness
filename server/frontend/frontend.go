package frontend

//go:generate go-bindata -pkg frontend -o frontend_gen.go build/bundled/src/...

import (
	"net/http"

	"github.com/elazarl/go-bindata-assetfs"
)

// FileSystem returns a filesystem for the embedded Polymer front-end.
func FileSystem() http.FileSystem {
	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "build/bundled/src"}
	return &binaryFileSystem{
		fs,
	}
}

type binaryFileSystem struct {
	fs http.FileSystem
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(name[1:])
}
