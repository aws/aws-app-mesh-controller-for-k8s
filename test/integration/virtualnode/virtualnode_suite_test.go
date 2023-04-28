package virtualnode_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVirtualnode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtualnode Suite")
}
