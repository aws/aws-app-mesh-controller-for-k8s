package fishapp_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFishApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FishApp Suite")
}
