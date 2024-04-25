package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/testground/sdk-go/run"
	runtime "github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
	ipfsclusterhelper "github.com/your/module/name/compose_file_generator"
	"github.com/your/module/name/monitor"
)

var runtimeMonitor *monitor.Monitor
var client sync.Client
var peerNum int64
var (
	enrolledState     = sync.State("peerEnrolled")
	testFinishedState = sync.State("testfinished")
	ctx               = context.Background()
)

// retrieve the sync service client, it has been initialized by the SDK.
// signal entry in the 'enrolled' state, and obtain a sequence number.
// wait for Peer 1 (bootstrap node) to be started.
// grab the first peer's IPFS Peer ID - we will bootstrap into Peer 1.
// blocking call - wait for peer to be boostrapped in ipfs
// Publish the IPFS Peer ID so other test nodes can reference it.
func bootstrapAllPeers(ctx context.Context, initCtx *run.InitContext, enrolledState sync.State, runenv *runtime.RunEnv) (*ipfsclusterhelper.IpfsClusterPeerHelper, error) {
	client = initCtx.SyncClient

	peerNum = client.MustSignalEntry(ctx, enrolledState)

	var peer1Id string = ""
	if peerNum > 1 {

		lastPeerBootstrappedState := sync.State(fmt.Sprintf("peer_%d_bootstrapped", 1))
		err := <-client.MustBarrier(ctx, lastPeerBootstrappedState, 1).C
		if err != nil {
			runenv.RecordMessage("Failure creating Compose File Generator")
			runenv.RecordFailure(err)
			return nil, err
		}

		targetTopic := sync.NewTopic(fmt.Sprintf("peer_%d_id", 1), reflect.TypeOf(peer1Id))
		peerChan := make(chan string)
		_, err = client.Subscribe(ctx, targetTopic, peerChan)
		if err != nil {
			runenv.RecordMessage("Failure subscribing to topic %s", fmt.Sprintf("peer_%d_id", 1))
			runenv.RecordFailure(err)
			return nil, err
		}
		peer1Id = <-peerChan
	}

	clusterHelper, err := ipfsclusterhelper.New(int(peerNum), runenv, peer1Id)
	if err != nil {
		runenv.RecordMessage("Failure creating IPFS Cluster Helper")
		runenv.RecordFailure(err)
		return nil, err
	}
	runenv.RecordMessage("Peer #%d Inititalizing", peerNum)
	go clusterHelper.LaunchNode()
	runenv.RecordMessage("Waiting for Peer ID...")

	peerID := <-*clusterHelper.PeerIdChannel
	runenv.RecordMessage("Peer %d initialized successfully with Peer ID: %s", peerNum, peerID)

	topic := fmt.Sprintf("peer_%d_id", peerNum)
	client.MustPublish(ctx, sync.NewTopic(topic, reflect.TypeOf(peerID)), peerID)
	time.Sleep(time.Second * 1)
	bootstrappedstate := sync.State(fmt.Sprintf("peer_%d_bootstrapped", peerNum))
	allBootstrappedState := sync.State("boostrapped_count")
	client.MustSignalEntry(ctx, bootstrappedstate)
	client.MustSignalEntry(ctx, allBootstrappedState)

	runenv.RecordMessage("Peer %d is waiting for all peers to bootstrap...", peerNum)
	err = <-client.MustBarrier(ctx, allBootstrappedState, runenv.TestInstanceCount).C
	if err != nil {
		runenv.RecordMessage("Failure waiting for all peers to bootstrap")
		runenv.RecordFailure(err)
		return nil, err
	}
	runenv.RecordMessage("Peer %d: All Peers Have Bootstrapped", peerNum)
	return clusterHelper, nil
}

func shutDownPeer(ctx context.Context, runenv *runtime.RunEnv, peerNum int64, clusterHelper *ipfsclusterhelper.IpfsClusterPeerHelper, client sync.Client) error {
	runenv.RecordMessage("Peer #%d is shutting down...", peerNum)
	err := clusterHelper.StopNode()
	if err != nil {
		runenv.RecordMessage("Error Stopping Peer %d", peerNum)
		runenv.RecordFailure(err)
		return err
	}

	shutdownState := sync.State("shutdownState")
	client.MustSignalEntry(ctx, shutdownState)
	return nil
}
