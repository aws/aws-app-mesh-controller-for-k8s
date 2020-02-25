## Kubernetes Requirements

* *Minimum supported Kubernetes version: v1.11*
* IAM permissions for the controller.  See the policy under "Using the install scripts" below, or create a cluster using eksctl with the following flags set:

```bash
eksctl create cluster --appmesh-access
```

Alternatively, you can use set up a role for the controller using [IAM for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).  This approach follows the principle of least privilege by keeping the permissions required by the controller off the worker node instance profile.

## Install

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
