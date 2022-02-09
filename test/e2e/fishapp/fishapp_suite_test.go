package fishapp_test

import (
	"testing"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var f *framework.Framework

func TestFishApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FishApp Suite")
}

var _ = BeforeSuite(func() {
	var err error
	f = framework.New(framework.GlobalOptions)
	Expect(err).NotTo(HaveOccurred())
})
