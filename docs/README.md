[![CircleCI](https://circleci.com/gh/aws/aws-app-mesh-controller-for-k8s/tree/master.svg?style=svg)](https://circleci.com/gh/aws/aws-app-mesh-controller-for-k8s/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/aws-app-mesh-controller-for-k8s)](https://goreportcard.com/report/github.com/aws/aws-app-mesh-controller-for-k8s) 

<p>
    <img src="assets/images/aws_appmesh_icon.svg" alt="App Mesh Logo" width="200" />
</p>

## AWS App Mesh Controller For K8s

AWS App Mesh Controller For K8s is a controller to help manage [App Mesh](https://aws.amazon.com/app-mesh/) resources for a Kubernetes cluster and injecting sidecars to Kubernetes [Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod/).  The controller watches custom resources for changes and reflects those changes into the [App Mesh API](https://docs.aws.amazon.com/app-mesh/latest/APIReference/Welcome.html). The controller maintains the custom resources ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)): meshes, virtualnodes, virtualrouters, virtualservices, virtualgateways and gatewayroutes.  The custom resources map to App Mesh API objects.

Note: For v0.5.0 or older versions of the controller, please refer to [legacy-controller branch](https://github.com/aws/aws-app-mesh-controller-for-k8s/tree/legacy-controller)

## Security disclosures

If you think you’ve found a potential security issue, please do not post it in the Issues.  Instead, please follow the instructions [here](https://aws.amazon.com/security/vulnerability-reporting/) or [email AWS security directly](mailto:aws-security@amazon.com).

## Documentation
Checkout our [Live Docs](https://aws.github.io/aws-app-mesh-controller-for-k8s/)!
