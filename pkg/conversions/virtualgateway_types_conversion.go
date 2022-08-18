package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_VirtualGatewayTLSValidationContextACMTrust_To_SDK_VirtualGatewayTLSValidationContextACMTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextACMTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust, scope conversion.Scope) error {
	sdkObj.CertificateAuthorityArns = aws.StringSlice(crdObj.CertificateAuthorityARNs)
	return nil
}

func Convert_CRD_VirtualGatewayTLSValidationContextFileTrust_To_SDK_VirtualGatewayTLSValidationContextFileTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextFileTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextFileTrust, scope conversion.Scope) error {
	sdkObj.CertificateChain = aws.String(crdObj.CertificateChain)
	return nil
}

func Convert_CRD_VirtualGatewayTLSValidationContextSDSTrust_To_SDK_VirtualGatewayTLSValidationContextSDSTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextSDSTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust, scope conversion.Scope) error {
	if crdObj.SecretName != nil {
		sdkObj.SecretName = crdObj.SecretName
	}
	return nil
}

func Convert_CRD_VirtualGatewayTLSValidationContextTrust_To_SDK_VirtualGatewayTLSValidationContextTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextTrust, scope conversion.Scope) error {
	if crdObj.ACM != nil {
		sdkObj.Acm = &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{}
		if err := Convert_CRD_VirtualGatewayTLSValidationContextACMTrust_To_SDK_VirtualGatewayTLSValidationContextACMTrust(crdObj.ACM, sdkObj.Acm, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Acm = nil
	}

	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{}
		if err := Convert_CRD_VirtualGatewayTLSValidationContextFileTrust_To_SDK_VirtualGatewayTLSValidationContextFileTrust(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}

	if crdObj.SDS != nil {
		sdkObj.Sds = &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{}
		if err := Convert_CRD_VirtualGatewayTLSValidationContextSDSTrust_To_SDK_VirtualGatewayTLSValidationContextSDSTrust(crdObj.SDS, sdkObj.Sds, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Sds = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayTLSValidationContextSubjectAlternativeNames_To_SDK_VirtualGatewayTLSValidationContextSubjectAlternativeNames(crdObj *appmesh.SubjectAlternativeNames, sdkObj *appmeshsdk.SubjectAlternativeNames, scope conversion.Scope) error {
	if crdObj.Match != nil {
		sdkObj.Match = &appmeshsdk.SubjectAlternativeNameMatchers{}
		sdkObj.Match.Exact = crdObj.Match.Exact
	}
	return nil
}

func Convert_CRD_VirtualGatewayTLSValidationContext_To_SDK_VirtualGatewayTLSValidationContext(crdObj *appmesh.VirtualGatewayTLSValidationContext, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContext, scope conversion.Scope) error {
	sdkObj.Trust = &appmeshsdk.VirtualGatewayTlsValidationContextTrust{}
	if err := Convert_CRD_VirtualGatewayTLSValidationContextTrust_To_SDK_VirtualGatewayTLSValidationContextTrust(&crdObj.Trust, sdkObj.Trust, scope); err != nil {
		return err
	}
	if crdObj.SubjectAlternativeNames != nil {
		sdkObj.SubjectAlternativeNames = &appmeshsdk.SubjectAlternativeNames{}
		if err := Convert_CRD_VirtualGatewayTLSValidationContextSubjectAlternativeNames_To_SDK_VirtualGatewayTLSValidationContextSubjectAlternativeNames(crdObj.SubjectAlternativeNames, sdkObj.SubjectAlternativeNames, scope); err != nil {
			return err
		}
	}
	return nil
}

func Convert_CRD_VirtualGatewayClientTLSCertificate_To_SDK_VirtualGatewayClientTLSCertificate(crdObj *appmesh.VirtualGatewayClientTLSCertificate, sdkObj *appmeshsdk.VirtualGatewayClientTlsCertificate, scope conversion.Scope) error {
	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{}
		if err := Convert_CRD_VirtualGatewayListenerTLSFileCertificate_To_SDK_VirtualGatewayListenerTLSFileCertificate(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}

	if crdObj.SDS != nil {
		sdkObj.Sds = &appmeshsdk.VirtualGatewayListenerTlsSdsCertificate{}
		if err := Convert_CRD_VirtualGatewayListenerTLSSDSCertificate_To_SDK_VirtualGatewayListenerTLSSDSCertificate(crdObj.SDS, sdkObj.Sds, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Sds = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayClientPolicyTLS_To_SDK_VirtualGatewayClientPolicyTLS(crdObj *appmesh.VirtualGatewayClientPolicyTLS, sdkObj *appmeshsdk.VirtualGatewayClientPolicyTls, scope conversion.Scope) error {
	sdkObj.Enforce = crdObj.Enforce

	var sdkPorts []*int64
	if len(crdObj.Ports) != 0 {
		sdkPorts = make([]*int64, 0, len(crdObj.Ports))
		for _, crdPort := range crdObj.Ports {
			sdkPorts = append(sdkPorts, aws.Int64((int64)(crdPort)))
		}
	}
	sdkObj.Ports = sdkPorts

	if crdObj.Certificate != nil {
		sdkObj.Certificate = &appmeshsdk.VirtualGatewayClientTlsCertificate{}
		if err := Convert_CRD_VirtualGatewayClientTLSCertificate_To_SDK_VirtualGatewayClientTLSCertificate(crdObj.Certificate, sdkObj.Certificate, scope); err != nil {
			return err
		}
	}

	sdkObj.Validation = &appmeshsdk.VirtualGatewayTlsValidationContext{}
	if err := Convert_CRD_VirtualGatewayTLSValidationContext_To_SDK_VirtualGatewayTLSValidationContext(&crdObj.Validation, sdkObj.Validation, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_VirtualGatewayClientPolicy_To_SDK_VirtualGatewayClientPolicy(crdObj *appmesh.VirtualGatewayClientPolicy, sdkObj *appmeshsdk.VirtualGatewayClientPolicy, scope conversion.Scope) error {
	if crdObj.TLS != nil {
		sdkObj.Tls = &appmeshsdk.VirtualGatewayClientPolicyTls{}
		if err := Convert_CRD_VirtualGatewayClientPolicyTLS_To_SDK_VirtualGatewayClientPolicyTLS(crdObj.TLS, sdkObj.Tls, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Tls = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayBackendDefaults_To_SDK_VirtualGatewayBackendDefaults(crdObj *appmesh.VirtualGatewayBackendDefaults, sdkObj *appmeshsdk.VirtualGatewayBackendDefaults, scope conversion.Scope) error {
	if crdObj.ClientPolicy != nil {
		sdkObj.ClientPolicy = &appmeshsdk.VirtualGatewayClientPolicy{}
		if err := Convert_CRD_VirtualGatewayClientPolicy_To_SDK_VirtualGatewayClientPolicy(crdObj.ClientPolicy, sdkObj.ClientPolicy, scope); err != nil {
			return err
		}
	} else {
		sdkObj.ClientPolicy = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayHealthCheckPolicy_To_SDK_VirtualGatewayHealthCheckPolicy(crdObj *appmesh.VirtualGatewayHealthCheckPolicy, sdkObj *appmeshsdk.VirtualGatewayHealthCheckPolicy, scope conversion.Scope) error {
	sdkObj.HealthyThreshold = aws.Int64(crdObj.HealthyThreshold)
	sdkObj.IntervalMillis = aws.Int64(crdObj.IntervalMillis)
	sdkObj.Path = crdObj.Path
	sdkObj.Port = (*int64)(crdObj.Port)
	sdkObj.Protocol = aws.String((string)(crdObj.Protocol))
	sdkObj.TimeoutMillis = aws.Int64(crdObj.TimeoutMillis)
	sdkObj.UnhealthyThreshold = aws.Int64(crdObj.UnhealthyThreshold)
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSACMCertificate_To_SDK_VirtualGatewayListenerTLSACMCertificate(crdObj *appmesh.VirtualGatewayListenerTLSACMCertificate, sdkObj *appmeshsdk.VirtualGatewayListenerTlsAcmCertificate, scope conversion.Scope) error {
	sdkObj.CertificateArn = aws.String(crdObj.CertificateARN)
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSFileCertificate_To_SDK_VirtualGatewayListenerTLSFileCertificate(crdObj *appmesh.VirtualGatewayListenerTLSFileCertificate, sdkObj *appmeshsdk.VirtualGatewayListenerTlsFileCertificate, scope conversion.Scope) error {
	sdkObj.CertificateChain = aws.String(crdObj.CertificateChain)
	sdkObj.PrivateKey = aws.String(crdObj.PrivateKey)
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSSDSCertificate_To_SDK_VirtualGatewayListenerTLSSDSCertificate(crdObj *appmesh.VirtualGatewayListenerTLSSDSCertificate, sdkObj *appmeshsdk.VirtualGatewayListenerTlsSdsCertificate, scope conversion.Scope) error {
	if crdObj.SecretName != nil {
		sdkObj.SecretName = crdObj.SecretName
	}
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSValidationContextFileTrust_To_SDK_VirtualGatewayListenerTLSValidationContextFileTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextFileTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextFileTrust, scope conversion.Scope) error {
	sdkObj.CertificateChain = aws.String(crdObj.CertificateChain)
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSValidationContextSDSTrust_To_SDK_VirtualGatewayListenerTLSValidationContextSDSTrust(crdObj *appmesh.VirtualGatewayTLSValidationContextSDSTrust, sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust, scope conversion.Scope) error {
	if crdObj.SecretName != nil {
		sdkObj.SecretName = crdObj.SecretName
	}
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSValidationContextTrust_To_SDK_VirtualGatewayListenerTLSValidationContextTrust(crdObj *appmesh.VirtualGatewayListenerTLSValidationContextTrust, sdkObj *appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust, scope conversion.Scope) error {
	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{}
		if err := Convert_CRD_VirtualGatewayListenerTLSValidationContextFileTrust_To_SDK_VirtualGatewayListenerTLSValidationContextFileTrust(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}

	if crdObj.SDS != nil {
		sdkObj.Sds = &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{}
		if err := Convert_CRD_VirtualGatewayListenerTLSValidationContextSDSTrust_To_SDK_VirtualGatewayListenerTLSValidationContextSDSTrust(crdObj.SDS, sdkObj.Sds, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Sds = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSValidationContextSubjectAlternativeNames_To_SDK_VirtualGatewayListenerTLSValidationContextSubjectAlternativeNames(crdObj *appmesh.SubjectAlternativeNames, sdkObj *appmeshsdk.SubjectAlternativeNames, scope conversion.Scope) error {
	if crdObj.Match != nil {
		sdkObj.Match.Exact = crdObj.Match.Exact
	}
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSValidationContext_To_SDK_VirtualGatewayListenerTLSValidationContext(crdObj *appmesh.VirtualGatewayListenerTLSValidationContext, sdkObj *appmeshsdk.VirtualGatewayListenerTlsValidationContext, scope conversion.Scope) error {
	sdkObj.Trust = &appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust{}
	if err := Convert_CRD_VirtualGatewayListenerTLSValidationContextTrust_To_SDK_VirtualGatewayListenerTLSValidationContextTrust(&crdObj.Trust, sdkObj.Trust, scope); err != nil {
		return err
	}
	if crdObj.SubjectAlternativeNames != nil {
		sdkObj.SubjectAlternativeNames = &appmeshsdk.SubjectAlternativeNames{}
		if err := Convert_CRD_VirtualGatewayListenerTLSValidationContextSubjectAlternativeNames_To_SDK_VirtualGatewayListenerTLSValidationContextSubjectAlternativeNames(crdObj.SubjectAlternativeNames, sdkObj.SubjectAlternativeNames, scope); err != nil {
			return err
		}
	}

	return nil
}

func Convert_CRD_VirtualGatewayListenerTLSCertificate_To_SDK_VirtualGatewayListenerTLSCertificate(crdObj *appmesh.VirtualGatewayListenerTLSCertificate, sdkObj *appmeshsdk.VirtualGatewayListenerTlsCertificate, scope conversion.Scope) error {
	if crdObj.ACM != nil {
		sdkObj.Acm = &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{}
		if err := Convert_CRD_VirtualGatewayListenerTLSACMCertificate_To_SDK_VirtualGatewayListenerTLSACMCertificate(crdObj.ACM, sdkObj.Acm, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Acm = nil
	}

	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{}
		if err := Convert_CRD_VirtualGatewayListenerTLSFileCertificate_To_SDK_VirtualGatewayListenerTLSFileCertificate(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}

	if crdObj.SDS != nil {
		sdkObj.Sds = &appmeshsdk.VirtualGatewayListenerTlsSdsCertificate{}
		if err := Convert_CRD_VirtualGatewayListenerTLSSDSCertificate_To_SDK_VirtualGatewayListenerTLSSDSCertificate(crdObj.SDS, sdkObj.Sds, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Sds = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayListenerTLS_To_SDK_VirtualGatewayListenerTLS(crdObj *appmesh.VirtualGatewayListenerTLS, sdkObj *appmeshsdk.VirtualGatewayListenerTls, scope conversion.Scope) error {
	sdkObj.Certificate = &appmeshsdk.VirtualGatewayListenerTlsCertificate{}
	if err := Convert_CRD_VirtualGatewayListenerTLSCertificate_To_SDK_VirtualGatewayListenerTLSCertificate(&crdObj.Certificate, sdkObj.Certificate, scope); err != nil {
		return err
	}

	if crdObj.Validation != nil {
		sdkObj.Validation = &appmeshsdk.VirtualGatewayListenerTlsValidationContext{}
		if err := Convert_CRD_VirtualGatewayListenerTLSValidationContext_To_SDK_VirtualGatewayListenerTLSValidationContext(crdObj.Validation, sdkObj.Validation, scope); err != nil {
			return err
		}
	}

	sdkObj.Mode = aws.String((string)(crdObj.Mode))
	return nil
}

func Convert_CRD_VirtualGatewayFileAccessLog_To_SDK_VirtualGatewayFileAccessLog(crdObj *appmesh.VirtualGatewayFileAccessLog, sdkObj *appmeshsdk.VirtualGatewayFileAccessLog, scope conversion.Scope) error {
	sdkObj.Path = aws.String(crdObj.Path)

	if crdObj.Format != nil {
		sdkObj.Format = &appmeshsdk.LoggingFormat{}
		Convert_CRD_LoggingFormat_To_SDK_LoggingFormat(crdObj.Format, sdkObj.Format)
	} else {
		sdkObj.Format = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayAccessLog_To_SDK_VirtualGatewayAccessLog(crdObj *appmesh.VirtualGatewayAccessLog, sdkObj *appmeshsdk.VirtualGatewayAccessLog, scope conversion.Scope) error {
	if crdObj.File != nil {
		sdkObj.File = &appmeshsdk.VirtualGatewayFileAccessLog{}
		if err := Convert_CRD_VirtualGatewayFileAccessLog_To_SDK_VirtualGatewayFileAccessLog(crdObj.File, sdkObj.File, scope); err != nil {
			return err
		}
	} else {
		sdkObj.File = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayLogging_To_SDK_VirtualGatewayLogging(crdObj *appmesh.VirtualGatewayLogging, sdkObj *appmeshsdk.VirtualGatewayLogging, scope conversion.Scope) error {
	if crdObj.AccessLog != nil {
		sdkObj.AccessLog = &appmeshsdk.VirtualGatewayAccessLog{}
		if err := Convert_CRD_VirtualGatewayAccessLog_To_SDK_VirtualGatewayAccessLog(crdObj.AccessLog, sdkObj.AccessLog, scope); err != nil {
			return err
		}
	} else {
		sdkObj.AccessLog = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewayPortMapping_To_SDK_VirtualGatewayPortMapping(crdObj *appmesh.VirtualGatewayPortMapping, sdkObj *appmeshsdk.VirtualGatewayPortMapping, scope conversion.Scope) error {
	sdkObj.Port = aws.Int64((int64)(crdObj.Port))
	sdkObj.Protocol = aws.String((string)(crdObj.Protocol))
	return nil
}

func Convert_CRD_VirtualGatewayHTTPConnectionPool_To_SDK_VirtualGatewayHttpConnectionPool(crdObj *appmesh.HTTPConnectionPool, sdkObj *appmeshsdk.VirtualGatewayHttpConnectionPool, scope conversion.Scope) error {
	sdkObj.MaxConnections = aws.Int64(crdObj.MaxConnections)
	sdkObj.MaxPendingRequests = crdObj.MaxPendingRequests
	return nil
}

func Convert_CRD_VirtualGatewayHTTP2ConnectionPool_To_SDK_VirtualGatewayHttp2ConnectionPool(crdObj *appmesh.HTTP2ConnectionPool, sdkObj *appmeshsdk.VirtualGatewayHttp2ConnectionPool, scope conversion.Scope) error {
	sdkObj.MaxRequests = aws.Int64(crdObj.MaxRequests)
	return nil
}

func Convert_CRD_VirtualGatewayGRPCConnectionPool_To_SDK_VirtualGatewayGrpcConnectionPool(crdObj *appmesh.GRPCConnectionPool, sdkObj *appmeshsdk.VirtualGatewayGrpcConnectionPool, scope conversion.Scope) error {
	sdkObj.MaxRequests = aws.Int64(crdObj.MaxRequests)
	return nil
}

func Convert_CRD_VirtualGatewayConnectionPool_To_SDK_VirtualGatewayConnectionPool(crdObj *appmesh.VirtualGatewayConnectionPool, sdkObj *appmeshsdk.VirtualGatewayConnectionPool, scope conversion.Scope) error {

	if crdObj.HTTP != nil {
		sdkObj.Http = &appmeshsdk.VirtualGatewayHttpConnectionPool{}
		if err := Convert_CRD_VirtualGatewayHTTPConnectionPool_To_SDK_VirtualGatewayHttpConnectionPool(crdObj.HTTP, sdkObj.Http, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http = nil
	}

	if crdObj.HTTP2 != nil {
		sdkObj.Http2 = &appmeshsdk.VirtualGatewayHttp2ConnectionPool{}
		if err := Convert_CRD_VirtualGatewayHTTP2ConnectionPool_To_SDK_VirtualGatewayHttp2ConnectionPool(crdObj.HTTP2, sdkObj.Http2, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http2 = nil
	}

	if crdObj.GRPC != nil {
		sdkObj.Grpc = &appmeshsdk.VirtualGatewayGrpcConnectionPool{}
		if err := Convert_CRD_VirtualGatewayGRPCConnectionPool_To_SDK_VirtualGatewayGrpcConnectionPool(crdObj.GRPC, sdkObj.Grpc, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Grpc = nil
	}

	return nil

}

func Convert_CRD_VirtualGatewayListener_To_SDK_VirtualGatewayListener(crdObj *appmesh.VirtualGatewayListener, sdkObj *appmeshsdk.VirtualGatewayListener, scope conversion.Scope) error {
	sdkObj.PortMapping = &appmeshsdk.VirtualGatewayPortMapping{}
	if err := Convert_CRD_VirtualGatewayPortMapping_To_SDK_VirtualGatewayPortMapping(&crdObj.PortMapping, sdkObj.PortMapping, scope); err != nil {
		return err
	}

	if crdObj.HealthCheck != nil {
		sdkObj.HealthCheck = &appmeshsdk.VirtualGatewayHealthCheckPolicy{}
		if err := Convert_CRD_VirtualGatewayHealthCheckPolicy_To_SDK_VirtualGatewayHealthCheckPolicy(crdObj.HealthCheck, sdkObj.HealthCheck, scope); err != nil {
			return err
		}
	} else {
		sdkObj.HealthCheck = nil
	}

	if crdObj.ConnectionPool != nil {
		sdkObj.ConnectionPool = &appmeshsdk.VirtualGatewayConnectionPool{}
		if err := Convert_CRD_VirtualGatewayConnectionPool_To_SDK_VirtualGatewayConnectionPool(crdObj.ConnectionPool, sdkObj.ConnectionPool, scope); err != nil {
			return err
		}
	} else {
		crdObj.ConnectionPool = nil
	}

	if crdObj.TLS != nil {
		sdkObj.Tls = &appmeshsdk.VirtualGatewayListenerTls{}
		if err := Convert_CRD_VirtualGatewayListenerTLS_To_SDK_VirtualGatewayListenerTLS(crdObj.TLS, sdkObj.Tls, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Tls = nil
	}
	return nil
}

func Convert_CRD_VirtualGatewaySpec_To_SDK_VirtualGatewaySpec(crdObj *appmesh.VirtualGatewaySpec, sdkObj *appmeshsdk.VirtualGatewaySpec, scope conversion.Scope) error {
	var sdkListeners []*appmeshsdk.VirtualGatewayListener
	if len(crdObj.Listeners) != 0 {
		sdkListeners = make([]*appmeshsdk.VirtualGatewayListener, 0, len(crdObj.Listeners))
		for _, crdListener := range crdObj.Listeners {
			sdkListener := &appmeshsdk.VirtualGatewayListener{}
			if err := Convert_CRD_VirtualGatewayListener_To_SDK_VirtualGatewayListener(&crdListener, sdkListener, scope); err != nil {
				return err
			}
			sdkListeners = append(sdkListeners, sdkListener)
		}
	}
	sdkObj.Listeners = sdkListeners

	if crdObj.Logging != nil {
		sdkObj.Logging = &appmeshsdk.VirtualGatewayLogging{}
		if err := Convert_CRD_VirtualGatewayLogging_To_SDK_VirtualGatewayLogging(crdObj.Logging, sdkObj.Logging, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Logging = nil
	}

	if crdObj.BackendDefaults != nil {
		sdkObj.BackendDefaults = &appmeshsdk.VirtualGatewayBackendDefaults{}
		if err := Convert_CRD_VirtualGatewayBackendDefaults_To_SDK_VirtualGatewayBackendDefaults(crdObj.BackendDefaults, sdkObj.BackendDefaults, scope); err != nil {
			return err
		}
	} else {
		sdkObj.BackendDefaults = nil
	}

	return nil
}
