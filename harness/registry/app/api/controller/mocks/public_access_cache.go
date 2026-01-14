//go:generate mockery --name RegistryMetadataHelper --output . --filename public_access_cache.go --outpkg mocks --with-expecter

package mocks

import (
	"context"

	"github.com/harness/gitness/registry/app/services/publicaccess"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

// MockPublicAccess is a mock implementation of the publicaccess.Service interface.
type MockPublicAccess struct {
	mock.Mock
}

// NewMockPublicAccess creates a new mock instance of PublicAccessService.
func NewMockPublicAccess() *MockPublicAccess {
	return &MockPublicAccess{}
}

// Get mocks the Get method.
func (m *MockPublicAccess) Get(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) (bool, error) {
	args := m.Called(ctx, resourceType, resourcePath)
	return args.Bool(0), args.Error(1)
}

// Set mocks the Set method.
func (m *MockPublicAccess) Set(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
	enable bool,
) error {
	args := m.Called(ctx, resourceType, resourcePath, enable)
	return args.Error(0)
}

// Delete mocks the Delete method.
func (m *MockPublicAccess) Delete(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) error {
	args := m.Called(ctx, resourceType, resourcePath)
	return args.Error(0)
}

// IsPublicAccessSupported mocks the IsPublicAccessSupported method.
func (m *MockPublicAccess) IsPublicAccessSupported(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	parentSpacePath string,
) (bool, error) {
	args := m.Called(ctx, resourceType, parentSpacePath)
	return args.Bool(0), args.Error(1)
}

// MarkChanged mocks the MarkChanged method.
func (m *MockPublicAccess) MarkChanged(
	ctx context.Context,
	publicAccessCacheKey *publicaccess.CacheKey,
) {
	m.Called(ctx, publicAccessCacheKey)
}

// ExpectGet sets up an expectation for the Get method.
func (m *MockPublicAccess) ExpectGet(
	ctx interface{},
	resourceType enum.PublicResourceType,
	resourcePath string,
	isPublic bool,
	err error,
) *mock.Call {
	return m.On("Get", ctx, resourceType, resourcePath).Return(isPublic, err)
}

// ExpectSet sets up an expectation for the Set method.
func (m *MockPublicAccess) ExpectSet(
	ctx interface{},
	resourceType enum.PublicResourceType,
	resourcePath string,
	enable bool,
	err error,
) *mock.Call {
	return m.On("Set", ctx, resourceType, resourcePath, enable).Return(err)
}

// ExpectDelete sets up an expectation for the Delete method.
func (m *MockPublicAccess) ExpectDelete(
	ctx interface{},
	resourceType enum.PublicResourceType,
	resourcePath string,
	err error,
) *mock.Call {
	return m.On("Delete", ctx, resourceType, resourcePath).Return(err)
}

// ExpectIsPublicAccessSupported sets up an expectation for the IsPublicAccessSupported method.
func (m *MockPublicAccess) ExpectIsPublicAccessSupported(
	ctx interface{},
	resourceType enum.PublicResourceType,
	parentSpacePath string,
	supported bool,
	err error,
) *mock.Call {
	return m.On("IsPublicAccessSupported", ctx, resourceType, parentSpacePath).Return(supported, err)
}

// ExpectMarkChanged sets up an expectation for the MarkChanged method.
func (m *MockPublicAccess) ExpectMarkChanged(
	ctx interface{},
	publicAccessCacheKey *publicaccess.CacheKey,
) *mock.Call {
	return m.On("MarkChanged", ctx, publicAccessCacheKey)
}
