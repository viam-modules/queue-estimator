package waitsensor

import (
	"fmt"
	"testing"

	"go.viam.com/test"
)

func makeValidConfig() Config {
	return Config{DetectorName: "test",
		CountPeriod: 5.0,
		NSamples:    10,
		CountThresholds: map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		ValidRegions: map[string][]BoundingBoxConfig{
			"box": {{XMin: 0.25,
				XMax: 0.75,
				YMin: 0.10,
				YMax: 0.90,
			}},
		},
	}
}

func TestValidateEmpty(t *testing.T) {
	var err error

	cfg := Config{}
	_, err = cfg.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "detector_name")

	missingName := makeValidConfig()
	missingName.DetectorName = ""
	_, err = missingName.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "detector_name")

	missingSamples := makeValidConfig()
	missingSamples.NSamples = 0.0
	_, err = missingSamples.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "n_samples")

	missingThresholds := makeValidConfig()
	missingThresholds.CountThresholds = nil
	_, err = missingThresholds.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "count_thresholds")

	missingRegions := makeValidConfig()
	missingRegions.ValidRegions = nil
	_, err = missingRegions.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "valid_regions")

}

func TestValidateValid(t *testing.T) {
	var err error

	validConfig := makeValidConfig()
	_, err = validConfig.Validate("")
	test.That(t, err, test.ShouldBeNil)

	// Note that you can have a 0-area bounding box! We'll just ignore it and use the whole image.
	validConfig.ValidRegions["box"][0].XMin = 0
	validConfig.ValidRegions["box"][0].XMax = 0
	_, err = validConfig.Validate("")
	test.That(t, err, test.ShouldBeNil)

	// and it's okay not to have a blank valid region!
	validConfig.ValidRegions["box"][0] = BoundingBoxConfig{}
	_, err = validConfig.Validate("")
	test.That(t, err, test.ShouldBeNil)

	// It's also okay to have a missing CountPeriod
	validConfig.CountPeriod = 0.0
	_, err = validConfig.Validate("")
	test.That(t, err, test.ShouldBeNil)
}

func TestValidateInvalidBoundingBox(t *testing.T) {
	var err error

	negativeXBounds := makeValidConfig()
	negativeXBounds.ValidRegions["box"][0].XMin = -0.2
	_, err = negativeXBounds.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be numbers between 0 and 1")

	negativeYBounds := makeValidConfig()
	negativeYBounds.ValidRegions["box"][0].YMin = -0.2
	_, err = negativeYBounds.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be numbers between 0 and 1")

	incorrectOrderX := makeValidConfig()
	incorrectOrderX.ValidRegions["box"][0].XMin = 0.99
	_, err = incorrectOrderX.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	fmt.Println(incorrectOrderX)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be less than x_max")

	incorrectOrderY := makeValidConfig()
	incorrectOrderY.ValidRegions["box"][0].YMin = 0.99
	_, err = incorrectOrderY.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be less than y_max")

	tooBigX := makeValidConfig()
	tooBigX.ValidRegions["box"][0].XMax = 2.0
	_, err = tooBigX.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be numbers between 0 and 1")

	tooBigY := makeValidConfig()
	tooBigY.ValidRegions["box"][0].YMax = 2.0
	_, err = tooBigY.Validate("")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "must be numbers between 0 and 1")
}
