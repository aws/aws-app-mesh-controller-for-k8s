module github.com/aws/aws-app-mesh-controller-for-k8s

go 1.13

require (
	github.com/aws/aws-sdk-go v1.25.19
	github.com/deckarep/golang-set v1.7.1
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/tools v0.0.0-20190710153321-831012c29e42 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20191025225708-5524a3672fbb
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/gengo v0.0.0-20190822140433-26a664648505 // indirect
	k8s.io/klog v1.0.0
)

// Kubernetes 1.15.0
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191025225708-5524a3672fbb
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/utils => k8s.io/utils v0.0.0-20191010214722-8d271d903fe4
)
