package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/testground/sdk-go/run"
	runtime "github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
	filegenerator "github.com/your/module/name/file_generator"
	ipfsclusterpeer "github.com/your/module/name/ipfs-cluster-peer"
	"github.com/your/module/name/monitor"
)

/*
*
Test1:

	Peer 1 will:
		- Wait to recieve file CID from Peer 2.
		-
*/
const (
	FailurePlanStep_InitiateShutdown = 1
	FailurePlanStep_ShutdownComplete = 2
	FailurePlanStep_InitiateStartup  = 3
	FailurePlanStep_StartupComplete  = 4
)

func mustBarrierWithFailureState(state sync.State, target int, duration time.Duration) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel() // Make sure to cancel the context to release resources

	// Create a channel to receive the signal
	signal := client.MustBarrier(ctx, state, target).C
	failure := client.MustBarrier(ctx, sync.State("failure"), 1).C
	// Wait for either the signal or timeout
	select {
	case <-signal:
		// Received signal, proceed with shutdown
		return nil
	case <-failure:
		// Timeout occurred, handle accordingly (e.g., log, return, etc.)
		// Here you can decide what to do if the timeout is reached before receiving the signal
		return fmt.Errorf("other node(s) have failed")
	}
}

func enterFailureStepPlan(runenv *runtime.RunEnv, peerNum int64, clusterHelper ipfsclusterpeer.ClusterPeer, client sync.Client) error {
	enterFailureState := sync.State(fmt.Sprintf("Peer%dShutdown", peerNum))
	// wait for peer # 1 to signal this peer to shut down
	maxTimeout := runtimeMonitor.MaxRuntime - runtimeMonitor.GetElapsedTime()
	err := mustBarrierWithFailureState(enterFailureState, FailurePlanStep_InitiateShutdown, maxTimeout)
	if err != nil {
		return err
	}
	// shut down the peer
	err = shutDownPeer(ctx, runenv, peerNum, clusterHelper, client)
	if err != nil {
		return err
	}
	// inform peer 1 that we have shut down
	client.SignalEntry(ctx, enterFailureState)
	runtimeMonitor.Log("Init Shutdown Complete")
	// wait for peer 1 to tell us to start the node again
	maxTimeout = runtimeMonitor.MaxRuntime - runtimeMonitor.GetElapsedTime()
	mustBarrierWithFailureState(enterFailureState, FailurePlanStep_InitiateStartup, maxTimeout)
	go clusterHelper.StartNode()
	time.Sleep(5 * time.Second)
	// inform peer 1 that we have started the node again.
	runtimeMonitor.Log("Init Startup Complete")
	client.SignalEntry(ctx, enterFailureState)
	return nil
}

func Test1(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	// var (
	// 	counter = runenv.R().Counter("fileGetSuccesses")
	// )
	// initialize the runtime monitor
	runtimeMonitor = monitor.NewMonitor(time.Duration(runenv.IntParam("maxRuntimeMinutes"))*time.Minute, runenv.BooleanParam("verbose"))
	// monitor thread keeps track of total program runtime.
	go runtimeMonitor.Start()

	clusterHelper, err := bootstrapAllPeers(ctx, initCtx, enrolledState, runenv)
	if err != nil {
		return err
	}
	if runenv.BooleanParam("tearDown") {
		defer clusterHelper.TearDown()
	}
	fileInsertedTopic := "fileInserted"
	// failureCounter := met.Counter("FileFailures")
	// successCounter := met.Counter("FileSuccesses")

	if peerNum == 1 {
		// wait for peer 2 to insert a file.
		var fileCID string
		fileCidChan := make(chan string)
		subsc := client.MustSubscribe(ctx, sync.NewTopic(fileInsertedTopic, reflect.TypeOf(fileCID)), fileCidChan)
		subsc.Done()
		fileCID = <-fileCidChan
		runtimeMonitor.Debug("File CID Retrieved %s", fileCID)

		// pull the file back down.
		// err = clusterHelper.GetFile(fileCID)
		// if err != nil {
		// 	client.SignalEntry(ctx, testFinishedState)
		// 	runtimeMonitor.FailWithError(fmt.Errorf("failed to initially retrieve the file %s: %s", fileCID, err))
		// 	return err
		// }

		if fileCID == "" {
			client.SignalEntry(ctx, sync.State("failure"))
			client.SignalEntry(ctx, testFinishedState)
			return err
		}

		for i := runenv.TestGroupInstanceCount; i > 1; i-- {
			enterFailureState := sync.State(fmt.Sprintf("Peer%dShutdown", i))
			_, err = client.SignalAndWait(ctx, enterFailureState, FailurePlanStep_ShutdownComplete)
			if err != nil {
				client.SignalEntry(ctx, sync.State("failure"))
				client.SignalEntry(ctx, testFinishedState)
				return err
			}
			if runtimeMonitor.IsCancel() {
				continue
			}
			runtimeMonitor.Log("Peer %d has shutdown... waiting 10 seconds", i)
			time.Sleep(10 * time.Second)
			err = clusterHelper.ClearIPFSCache()
			if err != nil {
				runtimeMonitor.Log("Error clearing ipfs cache: %s", err)
			}
			if runtimeMonitor.IsCancel() {
				continue
			}
			time.Sleep(10 * time.Second)
			err = clusterHelper.GetFile(fileCID)
			if err != nil {
				runtimeMonitor.Log("Error getting file %s", err)
			} else {
				runtimeMonitor.Log("Success getting file %s", fileCID)
			}
		}
		for i := runenv.TestGroupInstanceCount; i > 1; i-- {
			enterFailureState := sync.State(fmt.Sprintf("Peer%dShutdown", i))
			_, err = client.SignalAndWait(ctx, enterFailureState, FailurePlanStep_StartupComplete)

		}
		if err != nil {
			client.SignalEntry(ctx, testFinishedState)
			return err
		}

		_, err := client.SignalEntry(ctx, testFinishedState)
		if err != nil {
			return err
		}
		/**

		Given N Peers total:

		Peer 1 will:
		- Wait for the file to be inserted by peer #2.
		- Pull the file back down

		- Tell Peer #N to shutdown
		- Wait for Peer #N to shutdown
		- After Peer #N shutdown, Attempt to pull the file back down

		- Tell Peer #N-1 to shutdown
		- Wait for Peer #N-1 to shutdown
		- After Peer #N-1 shutdown, Attempt to pull the file back down

		...

		*/
	} else if peerNum == 2 {
		/**
		Peer 2 will:
		 - insert a file
		*/
		fileSizeMB := runenv.IntParam("fileSizeMB")
		fg := filegenerator.New()
		defer fg.TearDown()
		// Generate a random file with a name and size in MB
		fileName := fg.GenerateFilename()
		fileName, err := fg.GenerateFile(fileName, fileSizeMB)
		if err != nil {
			runenv.RecordFailure(fmt.Errorf("error generating file: %s", err))
			fileName = "default.txt"
		}
		start := time.Now()
		// insert the generated file
		runenv.RecordMessage("File %s:%dMB inserting...", fileName, fileSizeMB)
		ecfile, err := clusterHelper.PinFile(fileName)
		duration := time.Since(start)
		if err != nil || runtimeMonitor.IsCancel() {
			// don't block other peers on error states
			client.MustPublish(ctx, sync.NewTopic(fileInsertedTopic, reflect.TypeOf("")), "")
			return err
		} else {
			runenv.RecordMessage("File %s inserted successfully in %s", fileName, duration.String())
			// met.RecordPoint("FileSizeMb", float64(fileSizeMB))
			// met.RecordPoint("SuccessfulFileinsertTime", float64(duration.Milliseconds()))
			runtimeMonitor.Debug(ecfile.PrettyPrint())
		}
		if ecfile.CID == "" {
			client.MustPublish(ctx, sync.NewTopic(fileInsertedTopic, reflect.TypeOf("")), "")
			runtimeMonitor.Debug("File CID is BLANK")
			return fmt.Errorf("file CID is BLANK")
		}
		client.MustPublish(ctx, sync.NewTopic(fileInsertedTopic, reflect.TypeOf(ecfile.CID)), ecfile.CID)
	}
	if peerNum > 1 {
		err := enterFailureStepPlan(runenv, peerNum, clusterHelper, client)
		if err != nil {
			return fmt.Errorf("error in failure step plan: %s", err)
		}
		pos := client.MustSignalEntry(ctx, testFinishedState)
		runtimeMonitor.Debug("TestFinished State: Peer %d position: %d", peerNum, pos)
		// wait for all other peers to finish.
		<-client.MustBarrier(ctx, testFinishedState, runenv.TestGroupInstanceCount).C
	}
	// runenv.R().Gauge("TotalRuntime").Update(runtimeMonitor.GetElapsedTime().Seconds())
	// runenv.RecordSuccess()
	return nil
}
