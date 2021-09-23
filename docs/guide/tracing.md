# AppMesh Tracing Guide
AppMesh controller supports integration with multiple tracing solutions for data plane.

## AWS X-Ray
1. Enable X-Ray tracing for the App Mesh data plane
    ```sh
    helm upgrade -i appmesh-controller eks/appmesh-controller \
        --namespace appmesh-system \
        --set tracing.enabled=true \
        --set tracing.provider=x-ray
    ```
    The above configuration will inject the AWS X-Ray daemon sidecar in each pod scheduled to run on the mesh.

    You can optionally use a specific X-Ray image by setting the following flags in addition to the above:
    ```sh
        --set xray.image.repository=public.ecr.aws/xray/aws-xray-daemon \
        --set xray.image.tag=3.3.3
    ```

**Note**: You should restart all pods running inside the mesh after enabling tracing.

## Datadog tracing
1. Install the Datadog agent in the `appmesh-system` namespace
2. Enable Datadog Tracing for the App Mesh data plane
    ```sh
    helm upgrade -i appmesh-controller eks/appmesh-controller \
        --namespace appmesh-system \
        --set tracing.enabled=true \
        --set tracing.provider=datadog \
        --set tracing.address=datadog.appmesh-system \
        --set tracing.port=8126
    ```

**Note**: You should restart all pods running inside the mesh after enabling tracing.

## Jaeger tracing
Follow instructions in [appmesh-jaeger](https://github.com/aws/eks-charts/tree/master/stable/appmesh-jaeger) helm chart.

## Tips

### Tracing agents running as DaemonSets
For Jaeger and Datadog, running tracing agents as DaemonSets will need the `tracing.address` set to `status.hostIP` to use the node's IP.
To do this, use the flag below
   ```sh
   --set tracing.address=ref:status.hostIP
   ```
