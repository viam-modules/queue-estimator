// Package main is a module which serves the queue estimation
package main

import (
	"context"

	"github.com/viam-modules/queue-estimator/waitsensor"
	"go.viam.com/rdk/components/sensor"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("queue-estimator"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) (err error) {
	myMod, err := module.NewModuleFromArgs(ctx)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, sensor.API, waitsensor.Model)
	if err != nil {
		return err
	}

	err = myMod.Start(ctx)
	defer myMod.Close(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
