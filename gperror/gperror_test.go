package gperror_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudberrydb/gp-common-go-libs/gperror"
)

func TestGpError(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gperror Suite")
}

var _ = Describe("gperror", func() {
	var testErr *gperror.GpError

	BeforeEach(func() {
		testErr = &gperror.GpError{
			ErrorCode: gperror.ErrorCode(4321),
			Err:       errors.New("test-error"),
		}
	})

	Describe("Error", func() {
		It("returns a formatted string representation of the error", func() {
			Expect(testErr.Error()).To(Equal("ERROR[4321] test-error"))
		})
	})

	Describe("GetCode", func() {
		It("returns the error code", func() {
			Expect(testErr.GetCode()).To(Equal(gperror.ErrorCode(4321)))
		})
	})

	Describe("GetErr", func() {
		It("returns a string representation of the embedded error", func() {
			Expect(testErr.GetErr()).To(MatchError(errors.New("test-error")))
		})
	})

	Describe("New", func() {
		It("matches an independently created struct", func() {
			expectedErr := &gperror.GpError{
				ErrorCode: gperror.ErrorCode(9999),
				Err:       errors.New("unexpected error: some error"),
			}
			Expect(gperror.New(9999, "unexpected error: %s", "some error")).To(Equal(expectedErr))
		})
	})
})
