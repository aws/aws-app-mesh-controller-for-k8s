## Install

*Minimum supported Kubernetes version: v1.11*

Although the controller can be used independently from the aws-app-mesh-inject webhook, it is recommended that they be used together.  To Install the webhook, execute the following commands:

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
                "appmesh:DeleteRoute"
            ],
            "Resource": "*"
        }
    ]
}
```

Next, launch the controller:

```

```bash
curl https://raw.githubusercontent.com/aws/aws-app-mesh-controller-for-k8s/v0.1.2/deploy/all.yaml | kubectl apply -f -
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
