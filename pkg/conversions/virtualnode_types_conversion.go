package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_TLSValidationContextACMTrust_To_SDK_TLSValidationContextACMTrust(crdObj *appmesh.TLSValidationContextACMTrust, sdkObj *appmeshsdk.TlsValidationContextAcmTrust, scope conversion.Scope) error {
	sdkObj.CertificateAuthorityArns = aws.StringSlice(crdObj.CertificateAuthorityARNs)
	return nil
}

func Convert_CRD_TLSValidationContextFileTrust_To_SDK_TLSValidationContextFileTrust(crdObj *appmesh.TLSValidationContextFileTrust, sdkObj *appmeshsdk.TlsValidationContextFileTrust, scope conversion.Scope) error {
	sdkObj.CertificateChain = aws.String(crdObj.CertificateChain)
	return nil
}

func Convert_CRD_TLSValidationContextTrust_To_SDK_TLSValidationContextTrust(crdObj *appmesh.TLSValidationContextTrust, sdkObj *appmeshsdk.TlsValidationContextTrust, scope conversion.Scope) error {
	if crdObj.ACM != nil {
		sdkObj.Acm = &appmeshsdk.TlsValidationContextAcmTrust{}
		if err := Convert_CRD_TLSValidationContextACMTrust_To_SDK_TLSValidationContextACMTrust(crdObj.ACM, sdkObj.Acm, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Acm = nil
	}

	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.TlsValidationContextFileTrust{}
		if err := Convert_CRD_TLSValidationContextFileTrust_To_SDK_TLSValidationContextFileTrust(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}
	return nil
}

func Convert_CRD_TLSValidationContext_To_SDK_TLSValidationContext(crdObj *appmesh.TLSValidationContext, sdkObj *appmeshsdk.TlsValidationContext, scope conversion.Scope) error {
	sdkObj.Trust = &appmeshsdk.TlsValidationContextTrust{}
	if err := Convert_CRD_TLSValidationContextTrust_To_SDK_TLSValidationContextTrust(&crdObj.Trust, sdkObj.Trust, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_ClientPolicyTLS_To_SDK_ClientPolicyTLS(crdObj *appmesh.ClientPolicyTLS, sdkObj *appmeshsdk.ClientPolicyTls, scope conversion.Scope) error {
	sdkObj.Enforce = crdObj.Enforce

	var sdkPorts []*int64
	if len(crdObj.Ports) != 0 {
		sdkPorts = make([]*int64, 0, len(crdObj.Ports))
		for _, crdPort := range crdObj.Ports {
			sdkPorts = append(sdkPorts, aws.Int64((int64)(crdPort)))
		}
	}
	sdkObj.Ports = sdkPorts

	sdkObj.Validation = &appmeshsdk.TlsValidationContext{}
	if err := Convert_CRD_TLSValidationContext_To_SDK_TLSValidationContext(&crdObj.Validation, sdkObj.Validation, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_ClientPolicy_To_SDK_ClientPolicy(crdObj *appmesh.ClientPolicy, sdkObj *appmeshsdk.ClientPolicy, scope conversion.Scope) error {
	if crdObj.TLS != nil {
		sdkObj.Tls = &appmeshsdk.ClientPolicyTls{}
		if err := Convert_CRD_ClientPolicyTLS_To_SDK_ClientPolicyTLS(crdObj.TLS, sdkObj.Tls, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Tls = nil
	}
	return nil
}

func Convert_CRD_VirtualServiceBackend_To_SDK_VirtualServiceBackend(crdObj *appmesh.VirtualServiceBackend, sdkObj *appmeshsdk.VirtualServiceBackend, scope conversion.Scope) error {
	sdkObj.VirtualServiceName = aws.String("")
	if err := scope.Convert(&crdObj.VirtualServiceRef, sdkObj.VirtualServiceName, scope.Flags()); err != nil {
		return err
	}

	if crdObj.ClientPolicy != nil {
		sdkObj.ClientPolicy = &appmeshsdk.ClientPolicy{}
		if err := Convert_CRD_ClientPolicy_To_SDK_ClientPolicy(crdObj.ClientPolicy, sdkObj.ClientPolicy, scope); err != nil {
			return err
		}
	} else {
		sdkObj.ClientPolicy = nil
	}

	return nil
}

func Convert_CRD_Backend_To_SDK_Backend(crdObj *appmesh.Backend, sdkObj *appmeshsdk.Backend, scope conversion.Scope) error {
	sdkObj.VirtualService = &appmeshsdk.VirtualServiceBackend{}
	if err := Convert_CRD_VirtualServiceBackend_To_SDK_VirtualServiceBackend(&crdObj.VirtualService, sdkObj.VirtualService, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_BackendDefaults_To_SDK_BackendDefaults(crdObj *appmesh.BackendDefaults, sdkObj *appmeshsdk.BackendDefaults, scope conversion.Scope) error {
	if crdObj.ClientPolicy != nil {
		sdkObj.ClientPolicy = &appmeshsdk.ClientPolicy{}
		if err := Convert_CRD_ClientPolicy_To_SDK_ClientPolicy(crdObj.ClientPolicy, sdkObj.ClientPolicy, scope); err != nil {
			return err
		}
	} else {
		sdkObj.ClientPolicy = nil
	}
	return nil
}

func Convert_CRD_HealthCheckPolicy_To_SDK_HealthCheckPolicy(crdObj *appmesh.HealthCheckPolicy, sdkObj *appmeshsdk.HealthCheckPolicy, scope conversion.Scope) error {
	sdkObj.HealthyThreshold = aws.Int64(crdObj.HealthyThreshold)
	sdkObj.IntervalMillis = aws.Int64(crdObj.IntervalMillis)
	sdkObj.Path = crdObj.Path
	sdkObj.Port = (*int64)(crdObj.Port)
	sdkObj.Protocol = aws.String((string)(crdObj.Protocol))
	sdkObj.TimeoutMillis = aws.Int64(crdObj.TimeoutMillis)
	sdkObj.UnhealthyThreshold = aws.Int64(crdObj.UnhealthyThreshold)
	return nil
}

func Convert_CRD_ListenerTLSACMCertificate_To_SDK_ListenerTLSACMCertificate(crdObj *appmesh.ListenerTLSACMCertificate, sdkObj *appmeshsdk.ListenerTlsAcmCertificate, scope conversion.Scope) error {
	sdkObj.CertificateArn = aws.String(crdObj.CertificateARN)
	return nil
}

func Convert_CRD_ListenerTLSFileCertificate_To_SDK_ListenerTLSFileCertificate(crdObj *appmesh.ListenerTLSFileCertificate, sdkObj *appmeshsdk.ListenerTlsFileCertificate, scope conversion.Scope) error {
	sdkObj.CertificateChain = aws.String(crdObj.CertificateChain)
	sdkObj.PrivateKey = aws.String(crdObj.PrivateKey)
	return nil
}

func Convert_CRD_ListenerTLSCertificate_To_SDK_ListenerTLSCertificate(crdObj *appmesh.ListenerTLSCertificate, sdkObj *appmeshsdk.ListenerTlsCertificate, scope conversion.Scope) error {
	if crdObj.ACM != nil {
		sdkObj.Acm = &appmeshsdk.ListenerTlsAcmCertificate{}
		if err := Convert_CRD_ListenerTLSACMCertificate_To_SDK_ListenerTLSACMCertificate(crdObj.ACM, sdkObj.Acm, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Acm = nil
	}

	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.ListenerTlsFileCertificate{}
		if err := Convert_CRD_ListenerTLSFileCertificate_To_SDK_ListenerTLSFileCertificate(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}
	return nil
}

func Convert_CRD_ListenerTLS_To_SDK_ListenerTLS(crdObj *appmesh.ListenerTLS, sdkObj *appmeshsdk.ListenerTls, scope conversion.Scope) error {
	sdkObj.Certificate = &appmeshsdk.ListenerTlsCertificate{}
	if err := Convert_CRD_ListenerTLSCertificate_To_SDK_ListenerTLSCertificate(&crdObj.Certificate, sdkObj.Certificate, scope); err != nil {
		return err
	}
	sdkObj.Mode = aws.String((string)(crdObj.Mode))
	return nil
}

func Convert_CRD_ListenerTimeoutTcp_To_SDK_ListenerTimeoutTcp(crdObj *appmesh.TCPTimeout, sdkObj *appmeshsdk.TcpTimeout, scope conversion.Scope) error {
	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}
	return nil
}

func Convert_CRD_ListenerTimeoutHttp_To_SDK_ListenerTimeoutHttp(crdObj *appmesh.HTTPTimeout, sdkObj *appmeshsdk.HttpTimeout, scope conversion.Scope) error {
	if crdObj.PerRequest != nil {
		sdkObj.PerRequest = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.PerRequest, sdkObj.PerRequest, scope); err != nil {
			return err
		}
	} else {
		sdkObj.PerRequest = nil
	}

	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}
	return nil
}

func Convert_CRD_ListenerTimeoutGrpc_To_SDK_ListenerTimeoutGrpc(crdObj *appmesh.GRPCTimeout, sdkObj *appmeshsdk.GrpcTimeout, scope conversion.Scope) error {
	if crdObj.PerRequest != nil {
		sdkObj.PerRequest = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.PerRequest, sdkObj.PerRequest, scope); err != nil {
			return err
		}
	} else {
		sdkObj.PerRequest = nil
	}

	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}
	return nil
}

func Convert_CRD_ListenerTimeout_To_SDK_ListenerTimeout(crdObj *appmesh.ListenerTimeout, sdkObj *appmeshsdk.ListenerTimeout, scope conversion.Scope) error {
	if crdObj.TCP != nil {
		sdkObj.Tcp = &appmeshsdk.TcpTimeout{}
		if err := Convert_CRD_ListenerTimeoutTcp_To_SDK_ListenerTimeoutTcp(crdObj.TCP, sdkObj.Tcp, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Tcp = nil
	}

	if crdObj.HTTP != nil {
		sdkObj.Http = &appmeshsdk.HttpTimeout{}
		if err := Convert_CRD_ListenerTimeoutHttp_To_SDK_ListenerTimeoutHttp(crdObj.HTTP, sdkObj.Http, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http = nil
	}

	if crdObj.HTTP2 != nil {
		sdkObj.Http2 = &appmeshsdk.HttpTimeout{}
		if err := Convert_CRD_ListenerTimeoutHttp_To_SDK_ListenerTimeoutHttp(crdObj.HTTP2, sdkObj.Http2, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http2 = nil
	}

	if crdObj.GRPC != nil {
		sdkObj.Grpc = &appmeshsdk.GrpcTimeout{}
		if err := Convert_CRD_ListenerTimeoutGrpc_To_SDK_ListenerTimeoutGrpc(crdObj.GRPC, sdkObj.Grpc, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Grpc = nil
	}

	return nil
}

func Convert_CRD_Listener_To_SDK_Listener(crdObj *appmesh.Listener, sdkObj *appmeshsdk.Listener, scope conversion.Scope) error {
	sdkObj.PortMapping = &appmeshsdk.PortMapping{}
	if err := Convert_CRD_PortMapping_To_SDK_PortMapping(&crdObj.PortMapping, sdkObj.PortMapping, scope); err != nil {
		return err
	}
	if crdObj.HealthCheck != nil {
		sdkObj.HealthCheck = &appmeshsdk.HealthCheckPolicy{}
		if err := Convert_CRD_HealthCheckPolicy_To_SDK_HealthCheckPolicy(crdObj.HealthCheck, sdkObj.HealthCheck, scope); err != nil {
			return err
		}
	} else {
		sdkObj.HealthCheck = nil
	}
	if crdObj.TLS != nil {
		sdkObj.Tls = &appmeshsdk.ListenerTls{}
		if err := Convert_CRD_ListenerTLS_To_SDK_ListenerTLS(crdObj.TLS, sdkObj.Tls, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Tls = nil
	}
	return nil
}

func Convert_CRD_AWSCloudMapInstanceAttribute_To_SDK_AWSCloudMapInstanceAttribute(crdObj *appmesh.AWSCloudMapInstanceAttribute, sdkObj *appmeshsdk.AwsCloudMapInstanceAttribute, scope conversion.Scope) error {
	sdkObj.Key = aws.String(crdObj.Key)
	sdkObj.Value = aws.String(crdObj.Value)
	return nil
}

func Convert_CRD_AWSCloudMapServiceDiscovery_To_SDK_AWSCloudMapServiceDiscovery(crdObj *appmesh.AWSCloudMapServiceDiscovery, sdkObj *appmeshsdk.AwsCloudMapServiceDiscovery, scope conversion.Scope) error {
	sdkObj.NamespaceName = aws.String(crdObj.NamespaceName)
	sdkObj.ServiceName = aws.String(crdObj.ServiceName)

	var sdkAttributes []*appmeshsdk.AwsCloudMapInstanceAttribute
	if len(crdObj.Attributes) != 0 {
		sdkAttributes = make([]*appmeshsdk.AwsCloudMapInstanceAttribute, 0, len(crdObj.Attributes))
		for _, crdAttribute := range crdObj.Attributes {
			sdkAttribute := &appmeshsdk.AwsCloudMapInstanceAttribute{}
			if err := Convert_CRD_AWSCloudMapInstanceAttribute_To_SDK_AWSCloudMapInstanceAttribute(&crdAttribute, sdkAttribute, scope); err != nil {
				return err
			}
			sdkAttributes = append(sdkAttributes, sdkAttribute)
		}
	}
	sdkObj.Attributes = sdkAttributes
	return nil
}

func Convert_CRD_DNSServiceDiscovery_To_SDK_DNSServiceDiscovery(crdObj *appmesh.DNSServiceDiscovery, sdkObj *appmeshsdk.DnsServiceDiscovery, scope conversion.Scope) error {
	sdkObj.Hostname = aws.String(crdObj.Hostname)
	return nil
}

func Convert_CRD_ServiceDiscovery_To_SDK_ServiceDiscovery(crdObj *appmesh.ServiceDiscovery, sdkObj *appmeshsdk.ServiceDiscovery, scope conversion.Scope) error {
	if crdObj.AWSCloudMap != nil {
		sdkObj.AwsCloudMap = &appmeshsdk.AwsCloudMapServiceDiscovery{}
		if err := Convert_CRD_AWSCloudMapServiceDiscovery_To_SDK_AWSCloudMapServiceDiscovery(crdObj.AWSCloudMap, sdkObj.AwsCloudMap, scope); err != nil {
			return err
		}
	} else {
		sdkObj.AwsCloudMap = nil
	}

	if crdObj.DNS != nil {
		sdkObj.Dns = &appmeshsdk.DnsServiceDiscovery{}
		if err := Convert_CRD_DNSServiceDiscovery_To_SDK_DNSServiceDiscovery(crdObj.DNS, sdkObj.Dns, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Dns = nil
	}

	return nil
}

func Convert_CRD_FileAccessLog_To_SDK_FileAccessLog(crdObj *appmesh.FileAccessLog, sdkObj *appmeshsdk.FileAccessLog, scope conversion.Scope) error {
	sdkObj.Path = aws.String(crdObj.Path)
	return nil
}

func Convert_CRD_AccessLog_To_SDK_AccessLog(crdObj *appmesh.AccessLog, sdkObj *appmeshsdk.AccessLog, scope conversion.Scope) error {
	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.FileAccessLog{}
		if err := Convert_CRD_FileAccessLog_To_SDK_FileAccessLog(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}
	return nil
}

func Convert_CRD_Logging_To_SDK_Logging(crdObj *appmesh.Logging, sdkObj *appmeshsdk.Logging, scope conversion.Scope) error {
	if crdObj.AccessLog != nil {
		sdkObj.AccessLog = &appmeshsdk.AccessLog{}
		if err := Convert_CRD_AccessLog_To_SDK_AccessLog(crdObj.AccessLog, sdkObj.AccessLog, scope); err != nil {
			return err
		}
	} else {
		sdkObj.AccessLog = nil
	}
	return nil
}

func Convert_CRD_VirtualNodeSpec_To_SDK_VirtualNodeSpec(crdObj *appmesh.VirtualNodeSpec, sdkObj *appmeshsdk.VirtualNodeSpec, scope conversion.Scope) error {
	var sdkListeners []*appmeshsdk.Listener
	if len(crdObj.Listeners) != 0 {
		sdkListeners = make([]*appmeshsdk.Listener, 0, len(crdObj.Listeners))
		for _, crdListener := range crdObj.Listeners {
			sdkListener := &appmeshsdk.Listener{}
			if err := Convert_CRD_Listener_To_SDK_Listener(&crdListener, sdkListener, scope); err != nil {
				return err
			}
			sdkListeners = append(sdkListeners, sdkListener)
		}
	}
	sdkObj.Listeners = sdkListeners

	if crdObj.ServiceDiscovery != nil {
		sdkObj.ServiceDiscovery = &appmeshsdk.ServiceDiscovery{}
		if err := Convert_CRD_ServiceDiscovery_To_SDK_ServiceDiscovery(crdObj.ServiceDiscovery, sdkObj.ServiceDiscovery, scope); err != nil {
			return err
		}
	} else {
		sdkObj.ServiceDiscovery = nil
	}

	var sdkBackends []*appmeshsdk.Backend
	if len(crdObj.Backends) != 0 {
		sdkBackends = make([]*appmeshsdk.Backend, 0, len(crdObj.Backends))
		for _, crdBackend := range crdObj.Backends {
			sdkBackend := &appmeshsdk.Backend{}
			if err := Convert_CRD_Backend_To_SDK_Backend(&crdBackend, sdkBackend, scope); err != nil {
				return err
			}
			sdkBackends = append(sdkBackends, sdkBackend)
		}
	}
	sdkObj.Backends = sdkBackends

	if crdObj.BackendDefaults != nil {
		sdkObj.BackendDefaults = &appmeshsdk.BackendDefaults{}
		if err := Convert_CRD_BackendDefaults_To_SDK_BackendDefaults(crdObj.BackendDefaults, sdkObj.BackendDefaults, scope); err != nil {
			return err
		}
	} else {
		sdkObj.BackendDefaults = nil
	}

	if crdObj.Logging != nil {
		sdkObj.Logging = &appmeshsdk.Logging{}
		if err := Convert_CRD_Logging_To_SDK_Logging(crdObj.Logging, sdkObj.Logging, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Logging = nil
	}
	return nil
}
