package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/testground/sdk-go/run"
	runtime "github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
	filegenerator "github.com/your/module/name/file_generator"
	ipfsclusterpeer "github.com/your/module/name/ipfs-cluster-peer"
	"github.com/your/module/name/monitor"
)

/**
Test2:
	Peer #1 will:
		- Pin some files to the ipfs cluster
		- Inform other peers that files have been pinned, to now potentially enter failure states.
		- Wait for a few seconds to let the failure states kick in
		- Clear IPFS cache
		- Attempt to retrieve the files back.
	Other Peers will:
		- Wait until files have been uploaded
		- Potentially enter a failure state.
*/

const (
	Stage_WaitForFilesToinsert      = 1
	Stage_WaitForFilesToBeRetrieved = 2
)

// Peer 2...N logic
func simulateFailureStates(ctx context.Context, runenv *runtime.RunEnv, peerNum int64, shutdownProb float64, clusterHelper *ipfsclusterpeer.IpfsClusterPeer, client sync.Client) error {

	alreadyShutdown := false

	runenv.RecordMessage("Checking For Failure state...")
	if runtimeMonitor.IsCancel() {
		runenv.RecordMessage("Peer #%d: Max Runtime Exceed", peerNum)
		return nil
	}
	filesinsertedState := sync.State("filesinserted")
	if !alreadyShutdown && rand.Float64() < shutdownProb {
		alreadyShutdown = true
		err := shutDownPeer(ctx, runenv, peerNum, clusterHelper, client)
		if err != nil {
			return err
		}
		<-client.MustBarrier(ctx, filesinsertedState, Stage_WaitForFilesToBeRetrieved).C
		runenv.RecordMessage("Peer #%d is starting again...", peerNum)
		go clusterHelper.StartNode()
		time.Sleep(time.Second * 5)

	} else {
		<-client.MustBarrier(ctx, filesinsertedState, Stage_WaitForFilesToBeRetrieved).C
	}
	return nil
}

// Peer 1 logic
func runinsertQueryFilesTest(runenv *runtime.RunEnv, clusterHelper *ipfsclusterpeer.IpfsClusterPeer) {
	fg := filegenerator.New()
	maxFiles := runenv.IntParam("maxFiles")
	fileSizeMB := runenv.IntParam("fileSizeMB")
	totalFilesInserted := 0
	var insertedCids []string
	for i := 0; i < maxFiles; i++ {
		// Check if the maximum runtime has been exceeded
		if runtimeMonitor.IsCancel() {
			runenv.RecordMessage("Peer #%d: Max Runtime Exceed", clusterHelper.PeerNumber)
			break
		}
		// Generate a random file with a name and size in MB
		fileName := fg.GenerateFilename()
		fileName, err := fg.GenerateFile(fileName, fileSizeMB)
		if err != nil {
			runenv.RecordFailure(fmt.Errorf("error generating file: %s", err))
			break
		}
		start := time.Now()
		// insert the generated file
		runenv.RecordMessage("File %s:%dMB inserting...", fileName, fileSizeMB)
		ecfile, err := clusterHelper.PinFile(fileName)
		duration := time.Since(start)
		if err != nil {
			runenv.RecordMessage("error inserting file to cluster: %s... waiting 1 minute to try again", err)
			time.Sleep(1 * time.Minute)
		} else {
			runenv.RecordMessage("File %s inserted successfully in %s", fileName, duration.String())
			totalFilesInserted += 1
			if ecfile.CID != "" {
				insertedCids = append(insertedCids, ecfile.CID)
			} else {
				runtimeMonitor.Log("No file cid for %s", fileName)
			}
			time.Sleep(time.Duration(10 * time.Second))
		}

	}
	// if runtimeMonitor.IsDebug() {
	// 	err := clusterHelper.PrintPinnedFiles("/home/jake/pinned_files_pre_shutdown.csv")
	// 	if err != nil {
	// 		runtimeMonitor.Debug("Error Getting Pinned Files")
	// 	}
	// }
	filesinsertedState := sync.State("filesinserted")
	client.SignalEntry(ctx, filesinsertedState)

	time.Sleep(15 * time.Second)

	err := clusterHelper.ClearIPFSCache()
	if err != nil {
		runenv.RecordMessage("Error Clearing IPFS Cache: %s", err)
	}

	runtimeMonitor.Debug("Retrieving %d files", len(insertedCids))
	for _, cid := range insertedCids {
		time.Sleep(10 * time.Second)
		err = clusterHelper.GetFile(cid)
		if err != nil {
			runenv.RecordMessage("Error Retrieving File %s: %s", cid, err)
		} else {
			runenv.RecordMessage("Successfully Retrieved File %s", cid)
		}
		if runtimeMonitor.IsCancel() {
			runtimeMonitor.Debug("Max Runtime Exceeded: Cancelling File Retrieval")
			break
		}
	}

	// if runtimeMonitor.IsDebug() {
	// 	err := clusterHelper.PrintPinnedFiles("/home/jake/pinned_files_post_shutdown.csv")
	// 	if err != nil {
	// 		runtimeMonitor.Debug("Error Getting Pinned Files")
	// 	}
	// }
	client.SignalEntry(ctx, filesinsertedState)

}

func Test2(runenv *runtime.RunEnv, initCtx *run.InitContext) error {

	// initialize the runtime monitor
	runtimeMonitor = monitor.NewMonitor(time.Duration(runenv.IntParam("maxRuntimeMinutes"))*time.Minute, runenv.BooleanParam("verbose"))
	// monitor thread keeps track of total program runtime.
	go runtimeMonitor.Start()

	clusterHelper, err := bootstrapAllPeers(ctx, initCtx, enrolledState, runenv)
	if err != nil {
		return err
	}
	defer clusterHelper.TearDown()

	shutdownProb := runenv.FloatParam("shutdownProbability")
	runtimeMonitor.Debug(fmt.Sprintf("Peer %d Shutdown Probability %f", peerNum, shutdownProb))

	nonbootstrapPeersFinishedState := sync.State("nonbootstrappeersfinished")
	bootstrapPeerFinishedState := sync.State("bootstrappeersfinished")
	if peerNum != 1 {
		// Get the failure state entry cadence from testground params
		// Calculate the elapsed time
		// Check if the maximum runtime has been exceeded
		// Simulate node shutdown based on shutdownProbability.
		filesinsertedState := sync.State("filesinserted")
		<-client.MustBarrier(ctx, filesinsertedState, Stage_WaitForFilesToinsert).C
		err = simulateFailureStates(ctx, runenv, peerNum, shutdownProb, clusterHelper, client)
		<-client.MustBarrier(ctx, bootstrapPeerFinishedState, 1).C
		client.MustSignalEntry(ctx, nonbootstrapPeersFinishedState)
		if err != nil {
			runenv.RecordFailure(err)
			return err
		}
	} else {
		// wait for all other peers to finish stopping / starting
		// go testMetrics(runenv, clusterHelper)
		runinsertQueryFilesTest(runenv, clusterHelper)
		client.MustSignalEntry(ctx, bootstrapPeerFinishedState)
		runtimeMonitor.Debug("Finished Running insert Files Test")
		err := <-client.MustBarrier(ctx, nonbootstrapPeersFinishedState, runenv.TestInstanceCount-1).C
		if err != nil {
			runenv.RecordMessage("Failure waiting for all peers to bootstrap")
			runenv.RecordFailure(err)
			return err
		}
		runtimeMonitor.Debug("Peer %d: Test Complete", peerNum)
	}

	return nil
}
