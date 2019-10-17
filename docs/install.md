## Install

### Kubernetes Requirements

* *Minimum supported Kubernetes version: v1.11*
* IAM permissions for the controller.  See the policy under "Using the install scripts" below, or create a cluster using eksctl with the following flags set:

```bash
eksctl create cluster --appmesh-access
```

Alternatively, you can use set up a role for the controller using [IAM for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).  This approach follows the principle of least privilege by keeping the permissions required by the controller off the worker node instance profile.

### With Helm (Recommended)

To start, add the eks-charts helm repository:

```bash
helm repo add eks https://aws.github.io/eks-charts
```

Create the `appmesh-system` namespace:

```bash
kubectl create ns appmesh-system
```

Apply the CRDs:

```bash
kubectl apply -f https://raw.githubusercontent.com/aws/eks-charts/master/stable/appmesh-controller/crds/crds.yaml
```

Install the appmesh-controller:

```bash
helm upgrade -i appmesh-controller eks/appmesh-controller --namespace appmesh-system
```

Install the mutating admission webhook (aws-app-mesh-inject):

```bash
helm upgrade -i appmesh-inject eks/appmesh-inject \
--namespace appmesh-system \
--set mesh.create=true \
--set mesh.name=global
```

If you've installed the App Mesh controllers with scripts, you can switch to Helm by removing the controllers with:

```bash
# remove injector objects
kubectl delete ns appmesh-inject
kubectl delete ClusterRoleBinding aws-app-mesh-inject-binding
kubectl delete ClusterRole aws-app-mesh-inject-cr
kubectl delete  MutatingWebhookConfiguration aws-app-mesh-inject

# remove controller objects
kubectl delete ns appmesh-system
kubectl delete ClusterRoleBinding app-mesh-controller-binding
kubectl delete ClusterRole app-mesh-controller
Note that you shouldn't delete the App Mesh CRDs or the App Mesh custom resources (virtual nodes or services) in your cluster. Once you've removed the App Mesh controller and injector objects, you can proceed with the Helm installation as described above.
```

### Using the install scripts

```bash
# use `export MESH_NAME=color-mesh` to work with the example in this repository.
export MESH_NAME="<my-mesh-name>"
curl https://raw.githubusercontent.com/aws/aws-app-mesh-inject/master/scripts/install.sh | bash
```

This will launch the webhook into the appmesh-inject namespace. Now add the correct permissions to your worker nodes (or your pod identity solution, like kube2iam):

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "appmesh:DescribeMesh",
                "appmesh:DescribeVirtualNode",
                "appmesh:DescribeVirtualService",
                "appmesh:DescribeVirtualRouter",
                "appmesh:DescribeRoute",
                "appmesh:CreateMesh",
                "appmesh:CreateVirtualNode",
                "appmesh:CreateVirtualService",
                "appmesh:CreateVirtualRouter",
                "appmesh:CreateRoute",
                "appmesh:UpdateMesh",
                "appmesh:UpdateVirtualNode",
                "appmesh:UpdateVirtualService",
                "appmesh:UpdateVirtualRouter",
                "appmesh:UpdateRoute",
                "appmesh:ListMeshes",
                "appmesh:ListVirtualNodes",
                "appmesh:ListVirtualServices",
                "appmesh:ListVirtualRouters",
                "appmesh:ListRoutes",
                "appmesh:DeleteMesh",
                "appmesh:DeleteVirtualNode",
                "appmesh:DeleteVirtualService",
                "appmesh:DeleteVirtualRouter",
                "appmesh:DeleteRoute",
                "servicediscovery:CreateService",
                "servicediscovery:GetService",
                "servicediscovery:RegisterInstance",
                "servicediscovery:DeregisterInstance",
                "servicediscovery:ListInstances",
                "route53:GetHealthCheck",
                "route53:CreateHealthCheck",
                "route53:UpdateHealthCheck",
                "route53:ChangeResourceRecordSets",
                "route53:DeleteHealthCheck"
            ],
            "Resource": "*"
        }
    ]
}
```

Next, launch the controller:

```bash
kubectl apply -f https://raw.githubusercontent.com/aws/aws-app-mesh-controller-for-k8s/master/deploy/all.yaml
```

Make sure it's ready:

```bash
kubectl rollout status deployment app-mesh-controller -n appmesh-system
```

__IMPORTANT NOTE__: If you are using older versions (prior v0.1.2) of controller and have existing CRDs, then you need to update the virtualservice CRD specs to include virtualRouter and its listener as shown below.
```
spec:
  meshName: color-mesh
  routes:
  - http:
      action:
        weightedTargets:
        - virtualNodeName: colorteller.appmesh-demo
          weight: 1
```
should change to
```
spec:
  meshName: color-mesh
  virtualRouter:
    name: color-router
    listeners:
    - portMapping:
        port: 9080
        protocol: http
  routes:
  - http:
      action:
        weightedTargets:
        - virtualNodeName: colorteller.appmesh-demo
          weight: 1
```

Now try following along with [the example](example.md) to get a feel for how it works!
