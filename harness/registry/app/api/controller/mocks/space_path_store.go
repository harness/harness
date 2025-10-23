package mocks

import (
	"context"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

// SpacePathStore is a mock of SpacePathStore interface.
type SpacePathStore struct {
	mock.Mock
}

// FindByPath provides a mock function
func (m *SpacePathStore) FindByPath(ctx context.Context, path string) (*types.SpacePath, error) {
	ret := m.Called(ctx, path)

	var r0 *types.SpacePath
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.SpacePath); ok {
		r0 = rf(ctx, path)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.SpacePath)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindPrimaryBySpaceID provides a mock function
func (m *SpacePathStore) FindPrimaryBySpaceID(ctx context.Context, spaceID int64) (*types.SpacePath, error) {
	ret := m.Called(ctx, spaceID)

	var r0 *types.SpacePath
	if rf, ok := ret.Get(0).(func(context.Context, int64) *types.SpacePath); ok {
		r0 = rf(ctx, spaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.SpacePath)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, spaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InsertSegment provides a mock function
func (m *SpacePathStore) InsertSegment(ctx context.Context, segment *types.SpacePathSegment) error {
	ret := m.Called(ctx, segment)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.SpacePathSegment) error); ok {
		r0 = rf(ctx, segment)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePrimarySegment provides a mock function
func (m *SpacePathStore) DeletePrimarySegment(ctx context.Context, spaceID int64) error {
	ret := m.Called(ctx, spaceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, spaceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePathsAndDescendandPaths provides a mock function
func (m *SpacePathStore) DeletePathsAndDescendandPaths(ctx context.Context, spaceID int64) error {
	ret := m.Called(ctx, spaceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, spaceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
