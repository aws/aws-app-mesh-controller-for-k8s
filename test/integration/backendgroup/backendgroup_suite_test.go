package backendgroup_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBackendGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BackendGroup Suite")
}
