package waitsensor

import (
	"testing"

	"go.viam.com/test"
)

var validConfig = Config{DetectorName: "test",
	                     CountPeriod: 5.0,
	                     NSamples: 10,
	                     ValidRegions: map[string][]BoundingBoxConfig{
	                         "one": {XMin: 0.25,
	                                 XMin: 0.25,
	                                 YMax: 0.10,
	                                 YMax: 0.90,
	                                 },
	                     },
		        	 }

func TestValidateEmpty(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "detector_name")
}

func TestValidateCorrect(t *testing.T) {
	_, err := validConfig.Validate("")
	test.That(t, err, test.ShouldBeNil)
}

func TestValidateInvalidBoundingBox(t *testing.T) {
	negativeBounds := validConfig
	negativeBounds.ValidRegions["one"].XMin = -0.2
	_, err := cfg.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "detector_name")
}

