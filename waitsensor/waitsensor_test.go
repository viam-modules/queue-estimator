package waitsensor

import (
	"testing"

	"go.viam.com/test"
)

func TestValidate(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "detector_name")
}
