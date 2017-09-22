package kubernetes

import (
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
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
func (e *engine) Setup(c *backend.Config) error {

	// Create PVC

	return nil
}

// Start the pipeline step.
func (e *engine) Exec(s *backend.Step) error {

	// create job
	//		bind volumes
	//		set image
	//		set entrypoint
	//		set command

	return nil
}

// DEPRECATED
// Kill the pipeline step.
func (e *engine) Kill(s *backend.Step) error {
	var gracePeriodSeconds int64 = 5

	return e.client.CoreV1().Pods("default").Delete(s.Name, &metaV1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &metaV1.DeletePropagationBackground,
	})
}

// Wait for the pipeline step to complete and returns
// the completion results.
func (e *engine) Wait(s *backend.Step) (*backend.State, error) {

	// watch job
	//	onComplete, build state and return

	return nil, nil
}

// Tail the pipeline step logs.
func (e *engine) Tail(s *backend.Step) (io.ReadCloser, error) {
	pod, err := e.client.CoreV1().Pods("default").Get(s.Name, metaV1.GetOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return nil, err
	}

	return e.client.CoreV1().RESTClient().Get().
		Namespace("default").
		Name(pod.Name).
		Resource("pods").
		SubResource("log").
		VersionedParams(&v1.PodLogOptions{}, scheme.ParameterCodec).
		Stream()
}

// Destroy the pipeline environment.
func (e *engine) Destroy(c *backend.Config) error {
	var gracePeriodSeconds int64 = 0 // immediately

	for _, stage := range c.Stages {
		for _, step := range stage.Steps {
			e.client.CoreV1().Pods("default").Delete(step.Name, &metaV1.DeleteOptions{
				GracePeriodSeconds: &gracePeriodSeconds,
				PropagationPolicy:  &metaV1.DeletePropagationBackground,
			})
		}
	}

	// Delete PVC

	return nil
}
