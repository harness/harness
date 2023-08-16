// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ScmType defines the different SCM types supported for CI.
type ScmType string

func (ScmType) Enum() []interface{} { return toInterfaceSlice(scmTypes) }

var scmTypes = ([]ScmType{
	ScmTypeGitness,
	ScmTypeGithub,
	ScmTypeGitlab,
	ScmTypeUnknown,
})

const (
	ScmTypeUnknown ScmType = "UNKNOWN"
	ScmTypeGitness ScmType = "GITNESS"
	ScmTypeGithub  ScmType = "GITHUB"
	ScmTypeGitlab  ScmType = "GITLAB"
)
