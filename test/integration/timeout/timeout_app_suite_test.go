package timeout_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTimeoutApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timeout Suite")
}
