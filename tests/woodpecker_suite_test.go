package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWoodpeckerVerifier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Woodpecker Test Suite")
}
