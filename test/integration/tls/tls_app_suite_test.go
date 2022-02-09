package tls_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTLSApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TLS Suite")
}
