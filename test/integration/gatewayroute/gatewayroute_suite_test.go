package gatewayroute_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGatewayroute(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gatewayroute Suite")
}
