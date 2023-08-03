package enum

// ScmType defines the different types of principal types at harness.
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
