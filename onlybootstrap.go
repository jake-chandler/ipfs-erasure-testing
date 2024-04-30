package main

import (
	"time"

	"github.com/testground/sdk-go/run"
	runtime "github.com/testground/sdk-go/runtime"
	"github.com/your/module/name/monitor"
)

func OnlyBootstrap(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	// initialize the runtime monitor
	runtimeMonitor = monitor.NewMonitor(time.Duration(runenv.IntParam("maxRuntimeMinutes"))*time.Minute, runenv.BooleanParam("verbose"))
	// monitor thread keeps track of total program runtime.
	go runtimeMonitor.Start()
	guage := runenv.R().Gauge("NumPeers")
	clusterHelper, err := bootstrapAllPeers(ctx, initCtx, enrolledState, runenv)
	if err != nil {
		return err
	}
	if runenv.BooleanParam("tearDown") {
		defer clusterHelper.TearDown()
	}

	// using metrics in other test cases doesn't work currently
	// trying to figure out why
	guage.Update(float64(runenv.TestInstanceCount))

	return nil
}
