// Package main is a module which serves the queue estimation
package main

import (
	"github.com/viam-modules/queue-estimator/waitsensor"
	"go.viam.com/rdk/components/sensor"

	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
)

func main() {
	module.ModularMain(resource.APIModel{API: sensor.API, Model: waitsensor.Model})
}
