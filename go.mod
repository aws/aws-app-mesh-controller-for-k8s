module github.com/aws/aws-app-mesh-controller-for-k8s

go 1.13

require (
	github.com/aws/aws-sdk-go v1.30.7
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.4.3
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	gomodules.xyz/jsonpatch/v2 v2.0.1
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
