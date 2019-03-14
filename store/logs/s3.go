// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package logs

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/drone/drone/core"
)

// NewS3Env returns a new S3 log store.
func NewS3Env(bucket, prefix, endpoint string, pathStyle bool) core.LogStore {
	disableSSL := false

	if endpoint != "" {
		disableSSL = !strings.HasPrefix(endpoint, "https://")
	}

	return &s3store{
		bucket: bucket,
		prefix: prefix,
		session: session.Must(
			session.NewSession(&aws.Config{
				Endpoint:         aws.String(endpoint),
				DisableSSL:       aws.Bool(disableSSL),
				S3ForcePathStyle: aws.Bool(pathStyle),
			}),
		),
	}
}

// NewS3 returns a new S3 log store.
func NewS3(session *session.Session, bucket, prefix string) core.LogStore {
	return &s3store{
		bucket:  bucket,
		prefix:  prefix,
		session: session,
	}
}

type s3store struct {
	bucket  string
	prefix  string
	session *session.Session
}

func (s *s3store) Find(ctx context.Context, step int64) (io.ReadCloser, error) {
	svc := s3.New(s.session)
	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(step)),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *s3store) Create(ctx context.Context, step int64, r io.Reader) error {
	uploader := s3manager.NewUploader(s.session)
	input := &s3manager.UploadInput{
		ACL:    aws.String("private"),
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(step)),
		Body:   r,
	}
	_, err := uploader.Upload(input)
	return err
}

func (s *s3store) Update(ctx context.Context, step int64, r io.Reader) error {
	return s.Create(ctx, step, r)
}

func (s *s3store) Delete(ctx context.Context, step int64) error {
	svc := s3.New(s.session)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(step)),
	})
	return err
}

func (s *s3store) key(step int64) string {
	return path.Join("/", s.prefix, fmt.Sprint(step))
}
