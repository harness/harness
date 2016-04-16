package builtin

import (
	"path/filepath"

	"github.com/drone/drone/engine/compiler/parse"
)

type filterOp struct {
	visitor
	status   string
	branch   string
	event    string
	environ  string
	platform string
	matrix   map[string]string
}

// NewFilterOp returns a transformer that filters (ie removes) steps
// from the process based on conditional logic in the yaml.
func NewFilterOp(status, branch, event, env string, matrix map[string]string) Visitor {
	return &filterOp{
		status:  status,
		branch:  branch,
		event:   event,
		environ: env,
		matrix:  matrix,
	}
}

func (v *filterOp) VisitContainer(node *parse.ContainerNode) error {
	v.visitStatus(node)
	v.visitBranch(node)
	v.visitEvent(node)
	v.visitMatrix(node)
	v.visitPlatform(node)
	return nil
}

// visitStatus is a helpfer function that converts an on_change status
// filter to either success or failure based on the prior build status.
func (v *filterOp) visitStatus(node *parse.ContainerNode) {
	if len(node.Conditions.Status) == 0 {
		node.Conditions.Status = []string{"success"}
		return
	}
	for _, status := range node.Conditions.Status {
		if status != "change" && status != "changed" && status != "changes" {
			continue
		}
		var want []string
		switch v.status {
		case "success":
			want = append(want, "failure")
		case "failure", "error", "killed":
			want = append(want, "success")
		default:
			want = []string{"success", "failure"}
		}
		node.Conditions.Status = append(node.Conditions.Status, want...)
		break
	}
}

// visitBranch is a helper function that disables container steps when
// the branch conditions are not satisfied.
func (v *filterOp) visitBranch(node *parse.ContainerNode) {
	if len(node.Conditions.Branch) == 0 {
		return
	}
	for _, pattern := range node.Conditions.Branch {
		if ok, _ := filepath.Match(pattern, v.branch); ok {
			return
		}
	}
	node.Disabled = true
}

// visitEnvironment is a helper function that disables container steps
// when the deployment environment conditions are not satisfied.
func (v *filterOp) visitEnvironment(node *parse.ContainerNode) {
	if len(node.Conditions.Environment) == 0 {
		return
	}
	for _, pattern := range node.Conditions.Environment {
		if ok, _ := filepath.Match(pattern, v.environ); ok {
			return
		}
	}
	node.Disabled = true
}

// visitEvent is a helper function that disables container steps
// when the build event conditions are not satisfied.
func (v *filterOp) visitEvent(node *parse.ContainerNode) {
	if len(node.Conditions.Event) == 0 {
		return
	}
	for _, pattern := range node.Conditions.Event {
		if ok, _ := filepath.Match(pattern, v.event); ok {
			return
		}
	}
	node.Disabled = true
}

func (v *filterOp) visitMatrix(node *parse.ContainerNode) {
	for key, val := range node.Conditions.Matrix {
		if v.matrix[key] != val {
			node.Disabled = true
			break
		}
	}
}

// visitPlatform is a helper function that disables container steps
// when the build event conditions are not satisfied.
func (v *filterOp) visitPlatform(node *parse.ContainerNode) {
	if len(node.Conditions.Platform) == 0 {
		return
	}
	for _, pattern := range node.Conditions.Platform {
		if ok, _ := filepath.Match(pattern, v.platform); ok {
			return
		}
	}
	node.Disabled = true
}
