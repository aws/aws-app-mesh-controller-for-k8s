package appmeshinject

const (
	//Pod Annotations

	//AppMeshCNIAnnotation specifies that CNI will be used to configure traffic interception
	AppMeshCNIAnnotation = "appmesh.k8s.aws/appmeshCNI"
	//AppMeshCpuRequestAnnotation specifies the CPU requests for proxy
	AppMeshCpuRequestAnnotation = "appmesh.k8s.aws/cpuRequest"
	//AppMeshEgressIgnoredPortsAnnotation specifies the IPs that need to be ignored when intercepting traffic
	AppMeshEgressIgnoredIPsAnnotation = "appmesh.k8s.aws/egressIgnoredIPs"
	//AppMeshEgressIgnoredPortsAnnotation specifies the ports that need to ingored when intercepting traffic
	AppMeshEgressIgnoredPortsAnnotation = "appmesh.k8s.aws/egressIgnoredPorts"
	//AppMeshIgnoredGIDAnnotation specifies the GID used by proxy
	AppMeshIgnoredGIDAnnotation = "appmesh.k8s.aws/ignoredGID"
	//AppMeshIgnoredUIDAnnotation specifies the UID used by proxy
	AppMeshIgnoredUIDAnnotation = "appmesh.k8s.aws/ignoredUID"
	//AppMeshMemoryRequestAnnotation specifies the memory requests for proxy
	AppMeshMemoryRequestAnnotation = "appmesh.k8s.aws/memoryRequest"
	//AppMeshPortsAnnotation specifies the ports that proxy will forward traffic to. By default this is detected using the Pod ports.
	AppMeshPortsAnnotation = "appmesh.k8s.aws/ports"
	//AppMeshPreviewAnnotation specifies that proxy should use App Mesh preview endpoint
	AppMeshPreviewAnnotation = "appmesh.k8s.aws/preview"
	//AppMeshProxyEgressPortAnnotation specifies the port used by proxy for egress traffic (traffic originating from app container to external services). This is fixed to AppMeshProxyEgressPort
	AppMeshProxyEgressPortAnnotation = "appmesh.k8s.aws/proxyEgressPort"
	//AppMeshProxyIngressPortAnnotation specifies the port used by proxy for incoming traffic. This is fixed to AppMeshProxyIngressPort
	AppMeshProxyIngressPortAnnotation = "appmesh.k8s.aws/proxyIngressPort"
	//AppMeshSidecarInjectAnnotation specifies proxy should be injected for pod. Other systems can use this annotation on pod to determine if proxy is injected or not
	AppMeshSidecarInjectAnnotation = "appmesh.k8s.aws/sidecarInjectorWebhook"
	//AppMeshVirtualNodeNameAnnotation specifies the App Mesh VirtualNode used by proxy
	AppMeshVirtualNodeNameAnnotation = "appmesh.k8s.aws/virtualNode"

	//Pod Labels

	//FargateProfileLabel is added by fargate-scheduler when pod is running on AWS Fargate
	FargateProfileLabel = "eks.amazonaws.com/fargate-profile"

	// Fixed proxy container configuration required by App Mesh

	AppMeshProxyEgressPort  = "15001"
	AppMeshProxyIngressPort = "15000"
	AppMeshProxyUID         = "1337"
)
