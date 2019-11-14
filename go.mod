module github.com/aws/aws-app-mesh-controller-for-k8s

go 1.13

require (
	github.com/aws/aws-sdk-go v1.25.19
	github.com/deckarep/golang-set v1.7.1
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.3.0
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5
	golang.org/x/tools v0.0.0-20190710153321-831012c29e42 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20191010143144-fbf594f18f80
	k8s.io/apimachinery v0.0.0-20191014065749-fb3eea214746
	k8s.io/client-go v0.0.0-20191014070654-bd505ee787b2
	k8s.io/code-generator v0.0.0-20191003035328-700b1226c0bd
	k8s.io/klog v1.0.0
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20181025213731-e84da0312774
	golang.org/x/lint => golang.org/x/lint v0.0.0-20181217174547-8f45f776aaf1
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/sync => golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190209173611-3b5209105503
	golang.org/x/text => golang.org/x/text v0.3.1-0.20181227161524-e6919f6577db
	golang.org/x/time => golang.org/x/time v0.0.0-20161028155119-f51c12702a4d
	k8s.io/api => k8s.io/api v0.0.0-20191010143144-fbf594f18f80
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191014065749-fb3eea214746
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191014070654-bd505ee787b2
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191003035328-700b1226c0bd
)
