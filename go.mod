module github.com/aws/aws-app-mesh-controller-for-k8s

go 1.13

require (
	github.com/aws/aws-sdk-go v1.29.13
	github.com/deckarep/golang-set v1.7.1
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/goccy/go-yaml v1.4.3 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mikefarah/yq/v3 v3.0.0-20200304043226-a06320f13c07 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.7.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/spf13/cobra v0.0.6
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5
	go.uber.org/zap v1.10.0
	golang.org/x/exp v0.0.0-20200228211341-fcea875c7e85 // indirect
	golang.org/x/sys v0.0.0-20200317113312-5766fd39f98d // indirect
	golang.org/x/tools v0.0.0-20200316212524-3e76bee198d8 // indirect
	gonum.org/v1/gonum v0.7.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
	helm.sh/helm/v3 v3.1.2
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/cli-runtime v0.17.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.17.2
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
