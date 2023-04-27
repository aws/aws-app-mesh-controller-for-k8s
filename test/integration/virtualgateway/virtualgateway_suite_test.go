package virtualgateway_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVirtualgateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtualgateway Suite")
}
