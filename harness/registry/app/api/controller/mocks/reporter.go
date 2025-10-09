package mocks

import (
	"context"

	registryevents "github.com/harness/gitness/registry/app/events/artifact"

	"github.com/stretchr/testify/mock"
)

// Reporter is a mock implementation of registryevents.Reporter
type Reporter struct {
	mock.Mock
}

func NewReporter() *Reporter {
	return &Reporter{}
}

// Report provides a mock function
func (m *Reporter) Report(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// ArtifactCreated provides a mock function
func (m *Reporter) ArtifactCreated(ctx context.Context, payload *registryevents.ArtifactCreatedPayload) {
	m.Called(ctx, payload)
}

// ArtifactDeleted provides a mock function
func (m *Reporter) ArtifactDeleted(ctx context.Context, payload *registryevents.ArtifactDeletedPayload) {
	m.Called(ctx, payload)
}
