package kubernetes

import (
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type engine struct {
	client *kubernetes.Clientset
}

// New returns a new Kubernetes Engine.
func New(endpoint string, kubeconfigPath string) (backend.Engine, error) {
	config, err := clientcmd.BuildConfigFromFlags(endpoint, kubeconfigPath)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &engine{client: client}, nil
}

// Setup the pipeline environment.
func (e *engine) Setup(*backend.Config) error {
	// POST /api/v1/namespaces
	return nil
}

// Start the pipeline step.
func (e *engine) Exec(*backend.Step) error {
	// POST /api/v1/namespaces/{namespace}/pods
	return nil
}

// DEPRECATED
// Kill the pipeline step.
func (e *engine) Kill(*backend.Step) error {
	return nil
}

// Wait for the pipeline step to complete and returns
// the completion results.
func (e *engine) Wait(*backend.Step) (*backend.State, error) {
	// GET /api/v1/watch/namespaces/{namespace}/pods
	// GET /api/v1/watch/namespaces/{namespace}/pods/{name}
	return nil, nil
}

// Tail the pipeline step logs.
func (e *engine) Tail(*backend.Step) (io.ReadCloser, error) {
	// GET /api/v1/namespaces/{namespace}/pods/{name}/log
	return nil, nil
}

// Destroy the pipeline environment.
func (e *engine) Destroy(*backend.Config) error {
	// DELETE /api/v1/namespaces/{name}
	return nil
}
