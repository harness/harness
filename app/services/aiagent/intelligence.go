// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiagent

import (
	"context"

	capabilitiesctrl "github.com/harness/gitness/app/api/controller/capabilities"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/capabilities"
	aitypes "github.com/harness/gitness/types/aigenerate"
)

var _ Intelligence = GitnessIntelligence{}

// This interface serves are the single interface to provide AI use cases.
type Intelligence interface {
	StepIntelligence
	StageIntelligence
	PipelineIntelligence
}

type StepIntelligence interface {
	GeneratePipelineStep(
		ctx context.Context,
		req *aitypes.PipelineStepGenerateRequest) (*aitypes.PipelineStepGenerateResponse, error)
	UpdatePipelineStep(
		ctx context.Context,
		req *aitypes.PipelineStepUpdateRequest) (*aitypes.PipelineStepUpdateResponse, error)
}

type StageIntelligence interface {
	GeneratePipelineStage(
		ctx context.Context,
		req *aitypes.PipelineStageGenerateRequest) (*aitypes.PipelineStageGenerateResponse, error)
	UpdatePipelineStage(
		ctx context.Context,
		req *aitypes.PipelineStageUpdateRequest) (*aitypes.PipelineStageUpdateResponse, error)
}

type PipelineIntelligence interface {
	GeneratePipeline(
		ctx context.Context,
		req *aitypes.PipelineGenerateRequest) (*aitypes.PipelineGenerateResponse, error)
	UpdatePipeline(
		ctx context.Context,
		req *aitypes.PipelineUpdateResponse) (*aitypes.PipelineUpdateResponse, error)
}

type GitnessIntelligence struct {
	authorizer authz.Authorizer
	cr         *capabilities.Registry
	cc         *capabilitiesctrl.Controller
}

// UpdatePipeline implements Intelligence.
func (h GitnessIntelligence) UpdatePipeline(
	_ context.Context,
	_ *aitypes.PipelineUpdateResponse) (*aitypes.PipelineUpdateResponse, error) {
	panic("unimplemented")
}

// UpdatePipelineStage implements Intelligence.
func (h GitnessIntelligence) UpdatePipelineStage(
	_ context.Context,
	_ *aitypes.PipelineStageUpdateRequest) (*aitypes.PipelineStageUpdateResponse, error) {
	panic("unimplemented")
}

// UpdatePipelineStep implements Intelligence.
func (h GitnessIntelligence) UpdatePipelineStep(
	_ context.Context,
	_ *aitypes.PipelineStepUpdateRequest) (*aitypes.PipelineStepUpdateResponse, error) {
	panic("unimplemented")
}

// GeneratePipeline implements Intelligence.
func (h GitnessIntelligence) GeneratePipeline(
	_ context.Context,
	_ *aitypes.PipelineGenerateRequest) (*aitypes.PipelineGenerateResponse, error) {
	panic("unimplemented")
}

// GeneratePipelineStage implements Intelligence.
func (h GitnessIntelligence) GeneratePipelineStage(
	_ context.Context,
	_ *aitypes.PipelineStageGenerateRequest) (*aitypes.PipelineStageGenerateResponse, error) {
	panic("unimplemented")
}

// GeneratePipelineStep implements Intelligence.
func (h GitnessIntelligence) GeneratePipelineStep(
	_ context.Context,
	_ *aitypes.PipelineStepGenerateRequest) (*aitypes.PipelineStepGenerateResponse, error) {
	panic("unimplemented")
}
