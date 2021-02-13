package pcpcliwrap

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInternalSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PCP CLI Wrap Internal Suite")
}
