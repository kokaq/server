package control

import (
	"context"
	"fmt"
	"time"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/protocol/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ControlPlane struct {
	proto.UnimplementedKokaqControlPlaneServer
	RootDir string
	store   *ControlStore
}

func NewControlPlane(rootDirectory string, shardManagerAddress string) (*ControlPlane, error) {
	return &ControlPlane{
		RootDir: rootDirectory,
		store:   NewControlStore(rootDirectory, shardManagerAddress),
	}, nil
}

func (d *ControlPlane) AddNamespace(c context.Context, p *proto.KokaqNamespaceRequest) (*proto.KokaqNamespaceResponse, error) {
	q, err := d.AddQueue(c, &proto.KokaqQueueRequest{
		Namespace: p.Namespace,
		Queue:     ".Default",
		CreatedOn: timestamppb.Now(),
	})
	return &proto.KokaqNamespaceResponse{
		Namespace:       q.Request.Namespace,
		TotalQueueCount: 1,
		CreatedOn:       q.CreatedOn,
	}, err
}

func (d *ControlPlane) GetDataplane(c context.Context, p *proto.GetDataplaneRequest) (*proto.GetDataplaneResponse, error) {
	logger.ConsoleLog("INFO", "Received GetDataplane request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)

	address, _, found := d.store.GetDataPlaneAddress(p.Namespace, p.Queue)
	if !found {
		logger.ConsoleLog("ERROR", "Failed to get shard address for Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		return &proto.GetDataplaneResponse{
			Namespace: p.Namespace,
			Queue:     p.Queue,
			Address:   "",
		}, fmt.Errorf("failed to get shard address for namespace=%s, queue=%s", p.Namespace, p.Queue)
	}

	// Successfully resolved address
	logger.ConsoleLog("INFO", "Successfully resolved dataplane address: Namespace=%s, Queue=%s, Address=%s", p.Namespace, p.Queue, address)
	return &proto.GetDataplaneResponse{
		Namespace: p.Namespace,
		Queue:     p.Queue,
		Address:   address,
	}, nil
}

// GetQueue retrieves metadata for a specific queue from the assigned shard.
func (d *ControlPlane) GetQueue(c context.Context, p *proto.KokaqQueueRequest) (*proto.KokaqQueueResponse, error) {
	logger.ConsoleLog("INFO", "Fetching queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)

	_, internalAddress, found := d.store.GetDataPlaneAddress(p.Namespace, p.Queue)
	if !found {
		logger.ConsoleLog("ERROR", "Failed to get shard address for Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		return nil, fmt.Errorf("failed to get queue for namespace=%s, queue=%s", p.Namespace, p.Queue)
	}

	return d.getQueueFromShard(internalAddress, p.Namespace, p.Queue)
}

// AddQueue creates a new queue by requesting a shard assignment and sending a creation RPC.
func (d *ControlPlane) AddQueue(c context.Context, p *proto.KokaqQueueRequest) (*proto.KokaqQueueResponse, error) {
	logger.ConsoleLog("INFO", "Creating new queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)

	_, internalAddress, success, newCreated, shardId := d.store.GetOrAddDataPlaneAddress(p.Namespace, p.Queue)
	if !newCreated || !success {
		logger.ConsoleLog("ERROR", "Failed to create shard address for Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		return nil, fmt.Errorf("failed to create queue for namespace=%s, queue=%s", p.Namespace, p.Queue)
	}

	return d.newQueueFromShard(internalAddress, p.Namespace, p.Queue, shardId)
}

// ClearQueue removes all messages from the specified queue on the shard.
func (d *ControlPlane) ClearQueue(c context.Context, p *proto.KokaqQueueRequest) (*proto.StatusResponse, error) {
	logger.ConsoleLog("INFO", "Clearing queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)

	_, internalAddress, found := d.store.GetDataPlaneAddress(p.Namespace, p.Queue)
	if !found {
		logger.ConsoleLog("ERROR", "Failed to get shard address for Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		return nil, fmt.Errorf("failed to get queue for namespace=%s, queue=%s", p.Namespace, p.Queue)
	}

	if cleared, err := d.clearQueueFromShards(internalAddress, p.Namespace, p.Queue); !cleared || err != nil {
		logger.ConsoleLog("ERROR", "Clear operation failed: %v", err)
		return nil, fmt.Errorf("cannot clear queue")
	}

	return &proto.StatusResponse{Success: true}, nil
}

// DeleteQueue deletes the queue from the shard and optionally removes its index.
func (d *ControlPlane) DeleteQueue(c context.Context, p *proto.KokaqQueueRequest) (*proto.StatusResponse, error) {
	logger.ConsoleLog("INFO", "Deleting queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)

	_, internalAddress, found := d.store.GetDataPlaneAddress(p.Namespace, p.Queue)
	if !found {
		logger.ConsoleLog("ERROR", "Failed to get shard address for Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		return nil, fmt.Errorf("failed to get queue for namespace=%s, queue=%s", p.Namespace, p.Queue)
	}

	if deleted, err := d.deleteQueueFromShards(internalAddress, p.Namespace, p.Queue); !deleted || err != nil {
		logger.ConsoleLog("ERROR", "Delete operation failed: %v", err)
		return nil, fmt.Errorf("cannot delete queue")
	}

	if d.store.RemoveDataPlaneAddress(p.Namespace, p.Queue) {
		return &proto.StatusResponse{Success: true}, nil
	} else {
		return &proto.StatusResponse{Success: false}, nil
	}
}

// GetStats returns basic stats about the namespace (currently unimplemented).
func (d *ControlPlane) GetStats(c context.Context, p *proto.KokaqNamespaceRequest) (*proto.KokaqStatsResponse, error) {
	logger.ConsoleLog("WARN", "Stats not implemented for Namespace=%s", p.Namespace)
	return nil, fmt.Errorf("failed to connect to shard manager")
}

func (d *ControlPlane) newQueueFromShard(shardDataAddress string, namespace string, queue string, shardId uint64) (*proto.KokaqQueueResponse, error) {
	// Attempt to connect to the data server
	conn, err := grpc.NewClient(shardDataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to data server at %s: %v", shardDataAddress, err)
		return nil, fmt.Errorf("failed to connect to data server: %v", err)
	}
	defer conn.Close()

	// Set up gRPC client and timeout context
	dataClient := proto.NewKokaqDataPlaneClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prepare request to create a new queue on the shard
	req := &proto.KokaqNewQueueRequest{
		Request: &proto.KokaqQueueRequest{Queue: queue, Namespace: namespace},
		ShardId: shardId,
	}

	logger.ConsoleLog("INFO", "Sending NewQueue request to shard: Namespace=%s, Queue=%s, ShardId=%x", namespace, queue, shardId)

	// Perform RPC call to create the queue
	res, err := dataClient.New(ctx, req)
	if err != nil || res == nil {
		logger.ConsoleLog("ERROR", "New RPC failed: Namespace=%s, Queue=%s, ShardId=%x, Error=%v", namespace, queue, shardId, err)
		return nil, fmt.Errorf("new rpc failed: %v", err)
	}
	logger.ConsoleLog("INFO", "New queue successfully created: Namespace=%s, Queue=%s, ShardId=%x", namespace, queue, shardId)
	return res, nil
}
func (d *ControlPlane) getQueueFromShard(shardDataAddress string, namespace string, queue string) (*proto.KokaqQueueResponse, error) {

	// Attempt to connect to the data server at the given shard address
	conn, err := grpc.NewClient(shardDataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to data server at %s: %v", shardDataAddress, err)
		return nil, fmt.Errorf("failed to connect to data server: %v", err)
	}
	defer conn.Close()

	// Create data plane client and context with timeout
	dataClient := proto.NewKokaqDataPlaneClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prepare request to retrieve existing queue
	req := &proto.KokaqQueueRequest{
		Namespace: namespace,
		Queue:     queue,
	}

	logger.ConsoleLog("INFO", "Sending GetQueue request to shard: Namespace=%s, Queue=%s", namespace, queue)

	// Perform RPC call to retrieve the queue
	res, err := dataClient.Get(ctx, req)
	if err != nil || res == nil {
		logger.ConsoleLog("ERROR", "GET RPC failed: Namespace=%s, Queue=%s, Error=%v", namespace, queue, err)
		return nil, fmt.Errorf("GET RPC failed: %v", err)
	}

	logger.ConsoleLog("INFO", "Queue successfully retrieved from shard: Namespace=%s, Queue=%s", namespace, queue)
	return res, nil
}
func (d *ControlPlane) deleteQueueFromShards(shardDataAddress string, namespace string, queue string) (bool, error) {
	// Attempt to connect to the shard data server
	conn, err := grpc.NewClient(shardDataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to data server at %s: %v", shardDataAddress, err)
		return false, fmt.Errorf("failed to connect to data server: %v", err)
	}
	defer conn.Close()

	// Create the gRPC client and context
	dataClient := proto.NewKokaqDataPlaneClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prepare the request for deletion
	req := &proto.KokaqQueueRequest{
		Namespace: namespace,
		Queue:     queue,
	}

	logger.ConsoleLog("INFO", "Sending DeleteQueue request to shard: Namespace=%s, Queue=%s", namespace, queue)

	// Make the RPC call
	res, err := dataClient.Delete(ctx, req)
	if err != nil || res == nil || !res.Success {
		logger.ConsoleLog("ERROR", "Delete RPC failed: Namespace=%s, Queue=%s, Error=%v", namespace, queue, err)
		return false, fmt.Errorf("delete RPC failed: %v", err)
	}

	logger.ConsoleLog("INFO", "Queue deleted successfully from shard: Namespace=%s, Queue=%s", namespace, queue)
	return true, nil
}
func (d *ControlPlane) clearQueueFromShards(shardDataAddress string, namespace string, queue string) (bool, error) {
	// Establish connection to the shard data server
	conn, err := grpc.NewClient(shardDataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to data server at %s: %v", shardDataAddress, err)
		return false, fmt.Errorf("failed to connect to data server: %v", err)
	}
	defer conn.Close()

	// Prepare gRPC client and context
	dataClient := proto.NewKokaqDataPlaneClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Construct request
	req := &proto.KokaqQueueRequest{
		Namespace: namespace,
		Queue:     queue,
	}

	logger.ConsoleLog("INFO", "Sending ClearQueue request: Namespace=%s, Queue=%s", namespace, queue)

	// Perform Clear RPC
	res, err := dataClient.Clear(ctx, req)
	if err != nil || res == nil || !res.Success {
		logger.ConsoleLog("ERROR", "Clear RPC failed: Namespace=%s, Queue=%s, Error=%v", namespace, queue, err)
		return false, fmt.Errorf("clear RPC failed: %v", err)
	}

	logger.ConsoleLog("INFO", "Queue cleared successfully: Namespace=%s, Queue=%s", namespace, queue)
	return true, nil
}
