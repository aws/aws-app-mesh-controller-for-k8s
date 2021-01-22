package inject

const (
	//AppMeshCPURequestAnnotation specifies the CPU requests for proxy
	AppMeshCPURequestAnnotation = "appmesh.k8s.aws/cpuRequest"
	//AppMeshMemoryRequestAnnotation specifies the memory requests for proxy
	AppMeshMemoryRequestAnnotation = "appmesh.k8s.aws/memoryRequest"

	//AppMeshCPULimitAnnotation specifies the CPU limits for proxy
	AppMeshCPULimitAnnotation = "appmesh.k8s.aws/cpuLimit"
	//AppMeshMemoryLimitAnnotation specifies the memory limits for proxy
	AppMeshMemoryLimitAnnotation = "appmesh.k8s.aws/memoryLimit"

	// === begin proxy settings annotations ===
	//AppMeshCNIAnnotation specifies that CNI will be used to configure traffic interception
	AppMeshCNIAnnotation = "appmesh.k8s.aws/appmeshCNI"
	//AppMeshPortsAnnotation specifies the ports that proxy will forward traffic to. By default this is detected using the Pod ports.
	AppMeshPortsAnnotation = "appmesh.k8s.aws/ports"
	//AppMeshEgressIgnoredPortsAnnotation specifies the IPs that need to be ignored when intercepting traffic
	AppMeshEgressIgnoredIPsAnnotation = "appmesh.k8s.aws/egressIgnoredIPs"
	//AppMeshEgressIgnoredPortsAnnotation specifies the ports that need to ignored when intercepting traffic
	AppMeshEgressIgnoredPortsAnnotation = "appmesh.k8s.aws/egressIgnoredPorts"
	//AppMeshIgnoredGIDAnnotation specifies the GID used by proxy
	AppMeshIgnoredGIDAnnotation = "appmesh.k8s.aws/ignoredGID"
	//AppMeshIgnoredUIDAnnotation specifies the UID used by proxy
	AppMeshIgnoredUIDAnnotation = "appmesh.k8s.aws/ignoredUID"
	//AppMeshProxyEgressPortAnnotation specifies the port used by proxy for egress traffic (traffic originating from app container to external services). This is fixed to AppMeshProxyEgressPort
	AppMeshProxyEgressPortAnnotation = "appmesh.k8s.aws/proxyEgressPort"
	//AppMeshProxyIngressPortAnnotation specifies the port used by proxy for incoming traffic. This is fixed to AppMeshProxyIngressPort
	AppMeshProxyIngressPortAnnotation = "appmesh.k8s.aws/proxyIngressPort"
	// == end proxy settings annotations ===

	//AppMeshPreviewAnnotation specifies that proxy should use App Mesh preview endpoint
	AppMeshPreviewAnnotation = "appmesh.k8s.aws/preview"
	//AppMeshSidecarInjectAnnotation specifies proxy should be injected for pod. Other systems can use this annotation on pod to determine if proxy is injected or not
	AppMeshSidecarInjectAnnotation = "appmesh.k8s.aws/sidecarInjectorWebhook"
	//AppMeshSecretMountsAnnotation specifies the list of Secret that need to be mounted to the proxy as a volume
	AppMeshSecretMountsAnnotation = "appmesh.k8s.aws/secretMounts"
	//AppMeshGatewaySkipImageOverride specifies if Virtual Gateway sidecar image override needs to be skipped for customers
	//to use their own sidecare image for Virtual Gateway
	AppMeshGatewaySkipImageOverride = "appmesh.k8s.aws/virtualGatewaySkipImageOverride"
	//AppMeshSDSAnnotation is used if SDS is enabled at the controller level but needs to be disabled
	//for a particular VirtualNode.
	AppMeshSDSAnnotation = "appmesh.k8s.aws/sds"

	//Pod Labels

	//FargateProfileLabel is added by fargate-scheduler when pod is running on AWS Fargate
	FargateProfileLabel = "eks.amazonaws.com/fargate-profile"
)
