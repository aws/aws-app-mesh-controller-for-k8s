package sidecar_v1_22

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSidecarApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sidecar v1.22 Suite")
}
