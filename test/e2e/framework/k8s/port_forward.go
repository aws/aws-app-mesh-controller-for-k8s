package k8s

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
)

// NewPortForwarder return a new port forwarder.
// We can use portForward to get a TCP channel to pod port through APIServer.
func NewPortForwarder(ctx context.Context, restCfg *rest.Config, pod *corev1.Pod,
	ports []string, readyChan chan struct{}) (*portforward.PortForwarder, error) {
	cs, _ := kubernetes.NewForConfig(restCfg)
	req := cs.RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")
	url := req.URL()
	// The generated url above is incorrect, fix them below
	url.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)

	transport, upgrader, err := spdy.RoundTripperFor(restCfg)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	return portforward.New(dialer, ports, ctx.Done(), readyChan, nil, ginkgo.GinkgoWriter)
}
