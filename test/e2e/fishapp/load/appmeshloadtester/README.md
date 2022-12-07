# AppMesh K8s Performance Test Tool

## Initial Setup
1. Pull the Appmesh K8s Controller repo from https://github.com/aws/aws-app-mesh-controller-for-k8s
2. Follow the [live docs](https://aws.github.io/aws-app-mesh-controller-for-k8s/) Guide section to install
   1. "appmesh-controller" using helm chart
   2. "appmesh-prometheus" using helm chart
3. Follow [App Mesh with EKS—Base Deployment](https://github.com/aws/aws-app-mesh-examples/blob/main/walkthroughs/eks/base.md) to create EKS cluster using `eksctl`
4. Alternatively, [Getting started with AWS App Mesh and Kubernetes](https://docs.aws.amazon.com/app-mesh/latest/userguide/getting-started-kubernetes.html) describes how to install appmesh-controller and create EKS cluster using `eksctl`  in one place.

## Step 1: Prerequisites
All the scripts and configs related to the load test are under this directory:  `<CONTROLLER_PATH>/test/e2e/fishapp/load/appmeshloadtester`. We will run all the necessary commands for the load test from this directory. Following are few prerequisites before starting the load test:
1. Make sure you have the latest version of [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) installed.
2. This tests uses Python 3.9.6 and few libraries such as: boto3, pandas, requests, botocore, altair, matplotlib, csv, numpy etc. which you may need to install. 
3. Load test results will be stored into S3 bucket. So, in `scripts/constants.py` give your `S3_BUCKET` a unique name.
4. In case you get `AccessDeniedException` (or any kind of accessing AWS resource denied exception) while creating any AppMesh resources (e.g., VirtualNode), don't forget to authenticate with your AWS account.


## Step 2: Set Environment Variables
We need to set a few environment variables before starting the load tests.

```bash
export CONTROLLER_PATH=<Path to the controller directory e.g., /home/userName/workplace/appmesh-controller/aws-app-mesh-controller-for-k8s>
export CLUSTER_NAME=<Name of the EKS cluster, e.g., appmeshtest>
export KUBECONFIG=<If eksctl is used to create the cluster, the KUBECONFIG will look like: ~/.kube/eksctl/clusters/cluster-name>
export AWS_REGION=us-west-2
export VPC_ID=<VPC ID of the cluster, can be found using:  aws eks describe-cluster --name $CLUSTER_NAME | grep 'vpcId'>
```



## Step 3: Configuring the Load Test
All parameters of the mesh, load tests, metrics can be specified in `config.json`

`backends_map` -: The mapping from each Virtual Node to its backend Virtual Services. For each unique node name in `backends_map`, 
a VirtualNode, Deployment, Service and VirtualService (with its VirtualNode as its target) are created at runtime.

`load_tests` -: Array of different test configurations that need to be run on the mesh. `url` is the service endpoint that Fortio (load generator) should hit.

`metrics` -: Map of metric_name to the corresponding metric PromQL logic

## Step 4: Running the Load Test
Under the directory `CONTROLLER_PATH/test/e2e/fishapp/load/appmeshloadtester`, run the driver script using the below command -:
> sh scripts/driver.sh

The driver script will perform the following -:
1. Port-forward the Prometheus service to local
3. Run the Ginkgo test which is the entrypoint for our load Test
4. Kill the Prometheus port-forwarding after the load Test is done


## Step 5: Analyze the Results
All the test results are saved into `S3_BUCKET` which was specified in `scripts/constants.py`.    
Optionally, you can run the `analyze_test_results/analyze_load_test_data.py` to visualize the results. The `analyze_load_test_data.py` will first download all the load test results from the `S3_BUCKET` into `analyze_test_results\data` directory and 
plot a graph against the actual QPS (query per second) Fortio sends to the first VirtualNode vs the max memory consumed by the container of that VirtualNode.  

## Description of other files
`load_driver.py` -: Script which reads `config.json` and triggers load tests, reads metrics from PromQL and writes to S3. Called from within ginkgo

`fortio.yaml` -: Spec of the Fortio components which are created during runtime

`request_handler.py` and `request_handler_driver.sh` -: The custom service that runs in each of the pods to handle and route incoming requests according 
to the mapping in `backends_map` 

`configmap.yaml` -: ConfigMap spec to mount above request_handler* files into the cluster instead of creating Docker containers. Don't forget to use the absolute path of `request_handler_driver.sh`

`cluster.yaml` -: A sample EKS cluster config. This `cluster.yaml` can be used to create an EKS cluster by running `eksctl create cluster -f cluster.yaml`
