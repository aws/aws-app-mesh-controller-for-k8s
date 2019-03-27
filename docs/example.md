#Deploy an Example Application

After following the [install instructions](install.md), you can deploy an example application with:

    curl https://raw.githubusercontent.com/aws/aws-app-mesh-controller-for-k8s/v0.1.0/examples/color.yaml | kubectl apply -f -

Alternatively, if you have the repository checked out, you can launch the example application with:

    make example

Once everything is finished, we can see what was created.  Everything from the example is added to the `appmesh-demo` namespace. Let's see what's there:

    kubectl -n appmesh-demo get all

You should see a collection of virtual services, virtual nodes and a mesh, along with native Kubernetes deployments and services.

    NAME                                     READY   STATUS    RESTARTS   AGE
    pod/colorgateway-795b95fdb-vvvkm         2/2     Running   0          3h
    pod/colorteller-86664b5956-nngqg         2/2     Running   0          3h
    pod/colorteller-black-6787756c7b-sghj5   2/2     Running   0          3h
    pod/colorteller-blue-55d6f99dc6-7k5z8    2/2     Running   0          3h
    pod/colorteller-red-578866ffb-m54nd      2/2     Running   0          3h
    pod/curler-5b467f98bb-9wsn6              1/1     Running   0          3h
    NAME                        TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
    service/colorgateway        ClusterIP   10.100.156.153   <none>        9080/TCP   3h
    service/colorteller         ClusterIP   10.100.83.255    <none>        9080/TCP   3h
    service/colorteller-black   ClusterIP   10.100.40.66     <none>        9080/TCP   3h
    service/colorteller-blue    ClusterIP   10.100.121.78    <none>        9080/TCP   3h
    service/colorteller-red     ClusterIP   10.100.99.30     <none>        9080/TCP   3h
    NAME                                DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    deployment.apps/colorgateway        1         1         1            1           3h
    deployment.apps/colorteller         1         1         1            1           3h
    deployment.apps/colorteller-black   1         1         1            1           3h
    deployment.apps/colorteller-blue    1         1         1            1           3h
    deployment.apps/colorteller-red     1         1         1            1           3h
    deployment.apps/curler              1         1         1            1           3h
    NAME                                           DESIRED   CURRENT   READY   AGE
    replicaset.apps/colorgateway-795b95fdb         1         1         1       3h
    replicaset.apps/colorteller-86664b5956         1         1         1       3h
    replicaset.apps/colorteller-black-6787756c7b   1         1         1       3h
    replicaset.apps/colorteller-blue-55d6f99dc6    1         1         1       3h
    replicaset.apps/colorteller-red-578866ffb      1         1         1       3h
    replicaset.apps/curler-5b467f98bb              1         1         1       3h
    NAME                              AGE
    mesh.appmesh.k8s.aws/color-mesh   3h
    NAME                                                         AGE
    virtualnode.appmesh.k8s.aws/colorgateway-appmesh-demo        3h
    virtualnode.appmesh.k8s.aws/colorteller-appmesh-demo         3h
    virtualnode.appmesh.k8s.aws/colorteller-black-appmesh-demo   3h
    virtualnode.appmesh.k8s.aws/colorteller-blue-appmesh-demo    3h
    virtualnode.appmesh.k8s.aws/colorteller-red-appmesh-demo     3h
    NAME                                                                         AGE
    virtualservice.appmesh.k8s.aws/colorgateway.appmesh-demo.svc.cluster.local   3h
    virtualservice.appmesh.k8s.aws/colorteller.appmesh-demo.svc.cluster.local    3h

Verify App Mesh is working
```
kubectl run -n appmesh-demo -it curler --image=tutum/curl /bin/bash

for i in {1..100}; do curl colorgateway:9080/color; echo; done
# Expect to see even distribution of three colors
# {"color":"white", "stats": {"black":0.36,"blue":0.32,"white":0.32}}
```

Next update the traffic weight towards the colorteller backends. 

```bash
    kubectl edit VirtualService colorteller.appmesh-demo -n appmesh-demo
```
For example, change the traffic to only forward to black.
```bash
spec:
  meshName: color-mesh
  routes:
  - http:
      action:
        weightedTargets:
        - virtualNodeName: colorteller.appmesh-demo
          weight: 0
        - virtualNodeName: colorteller-blue
          weight: 0
        - virtualNodeName: colorteller-black.appmesh-demo
          weight: 1
```

And verify in curl the traffic move towards black.

To clean up
```bash
kubectl delete namespace appmesh-demo && kubectl delete mesh color-mesh && kubectl delete crd meshes.appmesh.k8s.aws && kubectl delete crd virtualnodes.appmesh.k8s.aws && kubectl delete crd virtualservices.appmesh.k8s.aws && kubectl delete namespace appmesh-inject && kubectl delete namespace appmesh-system

```

### More about the App Mesh custom resources
The services are required because we are using DNS based service discovery.  More methods of service discovery will be supported in the near future.  Let's take a look at a virtual node:
    kubectl -n appmesh-demo get virtualnode colorgateway-appmesh-demo

    apiVersion: appmesh.k8s.aws/v1beta1
    kind: VirtualNode
    metadata:
      annotations:
        kubectl.kubernetes.io/last-applied-configuration: |
          {"apiVersion":"appmesh.k8s.aws/v1beta1","kind":"VirtualNode","metadata":{"annotations":{},"name":"colorgateway-appmesh-demo","namespace":"appmesh-demo"},"spec":{"backends":[{"virtualService":{"virtualServiceName":"colorteller.appmesh-demo.svc.cluster.local"}}],"listeners":[{"portMapping":{"port":9080,"protocol":"http"}}],"meshName":"color-mesh","serviceDiscovery":{"dns":{"hostName":"colorgateway.appmesh-demo.svc.cluster.local"}}}}
      creationTimestamp: 2019-03-22T16:59:51Z
      finalizers:
      - virtualNodeDeletion.finalizers.appmesh.k8s.aws
      generation: 1
      name: colorgateway-appmesh-demo
      namespace: appmesh-demo
      resourceVersion: "1790196"
      selfLink: /apis/appmesh.k8s.aws/v1beta1/namespaces/appmesh-demo/virtualnodes/colorgateway-appmesh-demo
      uid: e848ef9d-4cc3-11e9-9e27-02f4bc929ee6
    spec:
      backends:
      - virtualService:
          virtualServiceName: colorteller.appmesh-demo.svc.cluster.local
      listeners:
      - portMapping:
          port: 9080
          protocol: http
      meshName: color-mesh
      serviceDiscovery:
        dns:
          hostName: colorgateway.appmesh-demo.svc.cluster.local
    status:
      conditions:
      - lastTransitionTime: 2019-03-22T16:59:51Z
        status: "True"
        type: VirtualNodeActive
