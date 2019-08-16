package logs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"cloud.google.com/go/storage"
	"github.com/drone/drone/core"
)

// NewGCSEnv returns a new GCS store.
func NewGCSEnv(bucketName string) core.LogStore {
	return &gcsStore{
		bucketName: bucketName,
		bucket:     nil,
	}
}

type gcsStore struct {
	bucketName string
	bucket     *storage.BucketHandle
}

func (g *gcsStore) Find(ctx context.Context, step int64) (io.ReadCloser, error) {
	err := g.getBucket(ctx)
	if err != nil {
		return nil, err
	}
	obj := g.bucket.Object(fmt.Sprintf("%d", step))
	return obj.NewReader(ctx)
}

func (g *gcsStore) Create(ctx context.Context, step int64, r io.Reader) error {
	err := g.getBucket(ctx)
	if err != nil {
		return err
	}
	obj := g.bucket.Object(fmt.Sprintf("%d", step))
	w := obj.NewWriter(ctx)
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	n, err := w.Write(buf)

	if err != nil {
		return err
	}

	if n != len(buf) {
		return fmt.Errorf("Write operation did not complete successfully")
	}

	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (g *gcsStore) Update(ctx context.Context, step int64, r io.Reader) error {
	return g.Create(ctx, step, r)
}

func (g *gcsStore) Delete(ctx context.Context, step int64) error {
	err := g.getBucket(ctx)
	if err != nil {
		return err
	}
	obj := g.bucket.Object(fmt.Sprintf("%d", step))
	return obj.Delete(ctx)
}

func (g *gcsStore) getBucket(ctx context.Context) error {
	if g.bucket != nil {
		return nil
	}
	client, err := storage.NewClient(ctx)

	if err != nil {
		return err
	}
	g.bucket = client.Bucket(g.bucketName)
	return nil
}
