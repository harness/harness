package gcs

import "golang.org/x/net/context"

type Bucket interface {
	Name() string
	CreateObject(context.Context, *CreateObjectRequest) (*Object, error)
	CopyObject(ctx context.Context, req *CopyObjectRequest) (o *Object, err error)
}

type Object struct {
}

type CreateObjectRequest struct {
}

type CopyObjectRequest struct {
}

type Int int
type Array []int
type Chan <-chan int
type Ptr *int
