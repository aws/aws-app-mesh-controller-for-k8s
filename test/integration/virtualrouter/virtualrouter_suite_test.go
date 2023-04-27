package virtualrouter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVirtualrouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtualrouter Suite")
}
