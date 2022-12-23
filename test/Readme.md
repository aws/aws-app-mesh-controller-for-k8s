## AWS-APPMESH-CONTROLLER-TESTS
---
The test folder consists of integration and e2e tests which can be run using the Ginkgo framework

### Types of Tests  
#### e2e tests  
These tests deploy a sample test app within appmesh and tests for end to end connectivity using following modes.  
1. DNS Service Discovery type 
2. CloudMap Service Discovery type 

You can trigger e2e tests with following ginkgo command (default: mTLS is disabled)
```
ginkgo -v -r test/e2e/ -- --cluster-kubeconfig=<absolute_path_kube_config_file> --cluster-name=<cluster_name> --aws-region=<region> --aws-vpc-id=<vpc_id>

Sample
ginkgo -v -r test/e2e/ -- --cluster-kubeconfig=/Users/xxxx/.kube/config --cluster-name=test-cluster --aws-region=us-west-2 --aws-vpc-id=vpc-0afa5f08378f21e50 
```

Run above command again by enabling mTLS sds based. Enable mTLS-sds based as below  
1. Enable SDS on the controller
```
 helm upgrade -i appmesh-controller eks/appmesh-controller --namespace appmesh-system --set sds.enabled=true
```
2. Set IsTLSEnabled to false and IsmTLSEnabled to true over [here](https://github.com/aws/aws-app-mesh-controller-for-k8s/blob/67d8cf133696c3d035b700659ad27050f4b80f52/test/e2e/fishapp/dynamic_stack_test.go#L39)  
```
stackPrototype = fishapp.DynamicStack{
				IsTLSEnabled: false,
				//Please set "enable-sds" to true in controller, prior to enabling this.
				//*TODO* Rename it to include SDS in it's name once we convert file based TLS test -> file based mTLS test.
				IsmTLSEnabled: true,
```

Rerun the above ginkgo command for e2e test with these settings. This will test connectivity with mTLS enabled  

**NOTE**  
For running e2e test on ARM64 based instances, change following [line](https://github.com/aws/aws-app-mesh-controller-for-k8s/blob/67d8cf133696c3d035b700659ad27050f4b80f52/test/e2e/fishapp/dynamic_stack.go#L46) to use an arm compatible image as shown below  
```
defaultHTTPProxyImage = "ghcr.io/abhinavsingh/proxy.py:v2.4.0b3.dev31.ga062f80-linux.arm64.v8"
```
As of today we do not have SDS based mTLS support on ARM since the spire agent and spire server images are not compatible with arm. We will update the images once we have arm support from spire.File based mTLS should work without any issues on ARM instances as well.  

#### integration tests  
These tests check creation/deletion of different appmesh components such as virtualgateway, virtualnode etc.
You can run the entire suite with following ginkgo command
```
ginkgo -v -r test/integration/ -- --cluster-kubeconfig=<absolute_path_kube_config_file> --cluster-name=<cluster_name> --aws-region=<region> --aws-vpc-id=<vpc_id>

Sample
ginkgo -v -r test/integration/ -- --cluster-kubeconfig=/Users/xxxx/.kube/config --cluster-name=test-cluster --aws-region=us-west-2 --aws-vpc-id=vpc-0afa5f08378f21e50 
```

You can also run tests for individual component as below  
```
ginkgo -v -r test/integration/<component_name>/ -- --cluster-kubeconfig=<absolute_path_kube_config_file> --cluster-name=<cluster_name> --aws-region=<region> --aws-vpc-id=<vpc_id>

Sample
ginkgo -v -r test/integration/virtualnode/ -- --cluster-kubeconfig=/Users/xxxx/.kube/config --cluster-name=test-cluster --aws-region=us-west-2 --aws-vpc-id=vpc-0afa5f08378f21e50
```

In case of failures, refer to [Troubleshooting](https://github.com/aws/aws-app-mesh-controller-for-k8s/blob/master/docs/guide/troubleshooting.md) guide.   


### integration test suite with kind

There is a test runner script, `scripts/test-with-kind.sh`, that can be used to run
an isolated suite of tests in a local kubernetes cluster. This sets up a local test
kubernetes cluster using kinD, which is able to run tests in isolation and without
a EKS cluster.

This tool is mostly for integration testing, but can be used on any linux machine. Refer
to the script documentation for [execution and environment details](https://github.com/aws/aws-app-mesh-controller-for-k8s/blob/master/scripts/test-with-kind.sh).

**Requirements:**

- The suite has only been tested on linux. It's possible that it could be made work on macOS, but
  it's not officially supported or verified.

- The test suite has several local tools that must be present for the suite to run. The 
  best reference is the [integration test setup for github](https://github.com/aws/aws-app-mesh-controller-for-k8s/blob/master/.github/actions/integration-test/action.yml).

- There must be environment credentials - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN` -
  for a valid role with permissions to `AWSCloudMapFullAccess`, `AWSAppMeshFullAccess`, and `AWSAppMeshEnvoyAccess`.
