package kubernetes

import (
	"io"
	"strings"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type engine struct {
	client       *kubernetes.Clientset
	namespace    string
	stroageClass string
}

// New returns a new Kubernetes Engine.
func New(endpoint, kubeconfigPath, namespace, storageClass string) (backend.Engine, error) {
	config, err := clientcmd.BuildConfigFromFlags(endpoint, kubeconfigPath)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &engine{
		client:       client,
		namespace:    namespace,
		stroageClass: storageClass,
	}, nil
}

// Setup the pipeline environment.
func (e *engine) Setup(c *backend.Config) error {

	// Create PVC
	e.client.Core().
		PersistentVolumeClaims(v1.NamespaceDefault).
		Create(&v1.PersistentVolumeClaim{
			Name:      c.Volumes[0].Name,
			Namespace: v1.NamespaceDefault,
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			},
		})

	return nil
}

// Start the pipeline step.
func (e *engine) Exec(s *backend.Step) error {
	_, err := e.client.
		Core().
		Pods(metav1.NamespaceDefault).
		Create(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dnsName(s.Name),
				Namespace: metav1.NamespaceDefault,
				Labels:    s.Labels,
				Annotations: map[string]string{
					"key": "value",
				},
			},
			Spec: v1.PodSpec{
				// Volumes: []v1.Volume{
				// 	v1.Volume{},
				// },
				Containers: []v1.Container{
					v1.Container{
						Name:       s.Alias,
						Image:      s.Image,
						Command:    s.Entrypoint,
						Args:       s.Command,
						WorkingDir: s.WorkingDir,
						Env:        mapToEnvVars(s.Environment),
						//VolumeMounts: []v1.VolumeMount{},
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
				// NodeSelector: map[string]string{
				// 	"key": "value",
				// },
			},
		})
	if err != nil {
		return err
	}

	return nil
}

// DEPRECATED
// Kill the pipeline step.
func (e *engine) Kill(s *backend.Step) error {
	var gracePeriodSeconds int64 = 5

	dpb := metav1.DeletePropagationBackground

	return e.client.
		CoreV1().
		Pods(e.namespace).
		Delete(dnsName(s.Name), &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriodSeconds,
			PropagationPolicy:  &dpb,
		})
}

// Wait for the pipeline step to complete and returns
// the completion results.
func (e *engine) Wait(s *backend.Step) (*backend.State, error) {
	finished := make(chan bool)

	var podUpdated = func(old interface{}, new interface{}) {
		pod := new.(*v1.Pod)
		if pod.Name == dnsName(s.Name) {
			switch pod.Status.Phase {
			case v1.PodSucceeded, v1.PodFailed:
				finished <- true
			}
		}
	}

	resyncPeriod := 5 * time.Minute
	si := informers.NewSharedInformerFactory(e.client, resyncPeriod)
	si.Core().
		V1().
		Pods().
		Informer().
		AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: podUpdated,
			},
		)
	si.Start(wait.NeverStop)

	<-finished

	pod, err := e.client.CoreV1().Pods(e.namespace).Get(dnsName(s.Name), metav1.GetOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return nil, err
	}

	bs := &backend.State{
		ExitCode:  int(pod.Status.ContainerStatuses[0].State.Terminated.ExitCode),
		Exited:    true,
		OOMKilled: false,
	}

	return bs, nil
}

// Tail the pipeline step logs.
func (e *engine) Tail(s *backend.Step) (io.ReadCloser, error) {
	var podReady = false

	for !podReady {
		pod, err := e.client.
			CoreV1().
			Pods(e.namespace).
			Get(dnsName(s.Name), metav1.GetOptions{
				IncludeUninitialized: true,
			})
		if err != nil {
			return nil, err
		}

		switch pod.Status.Phase {
		case v1.PodPending, v1.PodUnknown:
		default:
			podReady = true
		}
	}

	return e.client.CoreV1().RESTClient().Get().
		Namespace(e.namespace).
		Name(dnsName(s.Name)).
		Resource("pods").
		SubResource("log").
		VersionedParams(&v1.PodLogOptions{
			Follow: true,
		}, scheme.ParameterCodec).
		Stream()
}

// Destroy the pipeline environment.
func (e *engine) Destroy(c *backend.Config) error {
	var gracePeriodSeconds int64 = 0 // immediately

	dpb := metav1.DeletePropagationBackground

	for _, stage := range c.Stages {
		for _, step := range stage.Steps {
			e.client.
				CoreV1().
				Pods(e.namespace).
				Delete(dnsName(step.Name), &metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriodSeconds,
					PropagationPolicy:  &dpb,
				})
		}
	}

	// Delete PVC

	return nil
}

func mapToEnvVars(m map[string]string) []v1.EnvVar {
	var ev []v1.EnvVar
	for k, v := range m {
		ev = append(ev, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return ev
}

func dnsName(i string) string {
	return strings.Replace(i, "_", "-", -1)
}
