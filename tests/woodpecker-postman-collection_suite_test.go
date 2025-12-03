package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPostmanCollections(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Woodpecker Postman Collection Suite")
}
