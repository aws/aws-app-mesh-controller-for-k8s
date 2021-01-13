package inject

const (
	// AppMeshPrefix for Annotations
	AppMeshPrefix = "appmesh.k8s.aws"

	//AppMeshCPURequestAnnotation specifies the CPU requests for proxy
	AppMeshCPURequestAnnotation = AppMeshPrefix + "/cpuRequest"
	//AppMeshMemoryRequestAnnotation specifies the memory requests for proxy
	AppMeshMemoryRequestAnnotation = AppMeshPrefix + "/memoryRequest"

	//AppMeshCPULimitAnnotation specifies the CPU limits for proxy
	AppMeshCPULimitAnnotation = AppMeshPrefix + "/cpuLimit"
	//AppMeshMemoryLimitAnnotation specifies the memory limits for proxy
	AppMeshMemoryLimitAnnotation = AppMeshPrefix + "/memoryLimit"

	// === begin proxy settings annotations ===
	//AppMeshCNIAnnotation specifies that CNI will be used to configure traffic interception
	AppMeshCNIAnnotation = AppMeshPrefix + "/appmeshCNI"
	//AppMeshPortsAnnotation specifies the ports that proxy will forward traffic to. By default this is detected using the Pod ports.
	AppMeshPortsAnnotation = AppMeshPrefix + "/ports"
	//AppMeshEgressIgnoredPortsAnnotation specifies the IPs that need to be ignored when intercepting traffic
	AppMeshEgressIgnoredIPsAnnotation = AppMeshPrefix + "/egressIgnoredIPs"
	//AppMeshEgressIgnoredPortsAnnotation specifies the ports that need to ignored when intercepting traffic
	AppMeshEgressIgnoredPortsAnnotation = AppMeshPrefix + "/egressIgnoredPorts"
	//AppMeshIgnoredGIDAnnotation specifies the GID used by proxy
	AppMeshIgnoredGIDAnnotation = AppMeshPrefix + "/ignoredGID"
	//AppMeshIgnoredUIDAnnotation specifies the UID used by proxy
	AppMeshIgnoredUIDAnnotation = AppMeshPrefix + "/ignoredUID"
	//AppMeshProxyEgressPortAnnotation specifies the port used by proxy for egress traffic (traffic originating from app container to external services). This is fixed to AppMeshProxyEgressPort
	AppMeshProxyEgressPortAnnotation = AppMeshPrefix + "/proxyEgressPort"
	//AppMeshProxyIngressPortAnnotation specifies the port used by proxy for incoming traffic. This is fixed to AppMeshProxyIngressPort
	AppMeshProxyIngressPortAnnotation = AppMeshPrefix + "/proxyIngressPort"
	// == end proxy settings annotations ===

	//AppMeshPreviewAnnotation specifies that proxy should use App Mesh preview endpoint
	AppMeshPreviewAnnotation = AppMeshPrefix + "/preview"
	//AppMeshSidecarInjectAnnotation specifies proxy should be injected for pod. Other systems can use this annotation on pod to determine if proxy is injected or not
	AppMeshSidecarInjectAnnotation = AppMeshPrefix + "/sidecarInjectorWebhook"
	//AppMeshSecretMountsAnnotation specifies the list of Secret that need to be mounted to the proxy as a volume
	AppMeshSecretMountsAnnotation = AppMeshPrefix + "/secretMounts"
	//AppMeshGatewaySkipImageOverride specifies if Virtual Gateway sidecar image override needs to be skipped for customers
	//to use their own sidecare image for Virtual Gateway
	AppMeshGatewaySkipImageOverride = AppMeshPrefix + "/virtualGatewaySkipImageOverride"

	//Pod Labels

	//FargateProfileLabel is added by fargate-scheduler when pod is running on AWS Fargate
	FargateProfileLabel = "eks.amazonaws.com/fargate-profile"
)
