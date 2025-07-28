package control

import (
	"context"
	"fmt"
	"time"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/protocol/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ControlStore struct {
	ShardManagerAddress string
	AddressIndex        map[string]map[string]string
}

func NewControlStore(rootDirectory string, shardManagerAddress string) *ControlStore {
	return &ControlStore{
		ShardManagerAddress: shardManagerAddress,
		AddressIndex:        make(map[string]map[string]string, 0),
	}
}

func (d *ControlStore) GetDataPlaneAddress(namespace string, queue string) (string, bool) {
	logger.ConsoleLog("INFO", "Resolving shard address: Namespace=%s, Queue=%s", namespace, queue)
	// Check if address is already cached
	if shardAddress, exists := d.AddressIndex[namespace][queue]; exists {
		logger.ConsoleLog("INFO", "Shard address found in cache: Namespace=%s, Queue=%s, Address=%s", namespace, queue, shardAddress)
		return shardAddress, true
	}
	// No cached address — initiate RPC to shard manager
	logger.ConsoleLog("INFO", "Shard address not found in cache: Namespace=%s, Queue=%s", namespace, queue)

	var err error
	address, _, _, err := d.getOrAddDataPlaneAddressFromShardManager(namespace, queue, false)
	if err == nil {
		// Update address cache
		// Initialize map if needed and cache address
		if _, ok := d.AddressIndex[namespace]; !ok {
			d.AddressIndex[namespace] = make(map[string]string)
		}
		d.AddressIndex[namespace][queue] = address
		return d.AddressIndex[namespace][queue], true
	} else {
		logger.ConsoleLog("ERROR", "Cannot get data plane from shard manager: Namespace=%s, Queue=%s", namespace, queue)
		return "", false
	}
}

func (d *ControlStore) GetOrAddDataPlaneAddress(namespace string, queue string) (dataPlaneAddress string, success bool, newQueue bool, shardId uint64) {
	logger.ConsoleLog("INFO", "Resolving shard address: Namespace=%s, Queue=%s", namespace, queue)
	// Check if address is already cached
	if shardAddress, exists := d.AddressIndex[namespace][queue]; exists {
		logger.ConsoleLog("INFO", "Shard address found in cache: Namespace=%s, Queue=%s, Address=%s", namespace, queue, shardAddress)
		return shardAddress, true, false, 0
	}
	// No cached address — initiate RPC to shard manager
	logger.ConsoleLog("INFO", "Shard address not found in cache: Namespace=%s, Queue=%s", namespace, queue)

	var created bool
	var err error
	address, created, newShardId, err := d.getOrAddDataPlaneAddressFromShardManager(namespace, queue, true)
	if err == nil {
		// Update address cache
		// Initialize map if needed and cache address
		if _, ok := d.AddressIndex[namespace]; !ok {
			d.AddressIndex[namespace] = make(map[string]string, 0)
		}
		d.AddressIndex[namespace][queue] = address
		return d.AddressIndex[namespace][queue], true, created, newShardId
	} else {
		logger.ConsoleLog("ERROR", "Cannot get data plane from shard manager: Namespace=%s, Queue=%s", namespace, queue)
		return "", false, false, 0
	}
}

func (d *ControlStore) RemoveDataPlaneAddress(namespace string, queue string) (success bool) {
	logger.ConsoleLog("INFO", "Cleaing shard address: Namespace=%s, Queue=%s", namespace, queue)
	// Check if address is already cached
	if shardAddress, exists := d.AddressIndex[namespace][queue]; exists {
		logger.ConsoleLog("INFO", "Shard address found in cache: Namespace=%s, Queue=%s, Address=%s", namespace, queue, shardAddress)
		delete(d.AddressIndex[namespace], queue)
	}

	conn, err := d.getShardManagerConnection()
	if err != nil {
		logger.ConsoleLog("ERROR", "Cannot reach shard manager to remove shard: Namespace=%s, Queue=%s", namespace, queue)
		return false
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Build GetShard request
	req := &proto.GetShardRequest{
		Namespace: namespace,
		Queue:     queue,
	}

	// Perform RPC to get shard assignment
	_, err = shardManagerClient.DeleteShard(ctx, req)
	if err != nil {
		logger.ConsoleLog("ERROR", "Shard manager cannot remove shard: Namespace=%s, Queue=%s: %v", namespace, queue, err)
		return false
	}
	return true
}

func (d *ControlStore) getOrAddDataPlaneAddressFromShardManager(namespace string, queue string, createIfNotFound bool) (string, bool, uint64, error) {
	logger.ConsoleLog("INFO", "Shard address not found in cache: Namespace=%s, Queue=%s. Contacting shard manager...", namespace, queue)

	conn, err := d.getShardManagerConnection()
	if err != nil {
		return "", false, 0, err
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Build GetShard request
	req := &proto.GetShardRequest{
		Namespace:        namespace,
		Queue:            queue,
		CreateIfNotFound: createIfNotFound,
	}

	// Perform RPC to get shard assignment
	res, err := shardManagerClient.GetShard(ctx, req)
	if err != nil {
		logger.ConsoleLog("ERROR", "GetShard RPC failed: Namespace=%s, Queue=%s: %v", namespace, queue, err)
		return "", false, 0, fmt.Errorf("shardmanager.getShard rpc failed: %v", err)
	}

	// Validate response
	if res != nil && res.GrpcAddress != "" {
		if res.IsNew {
			logger.ConsoleLog("INFO", "New shard created: Namespace=%s, Queue=%s, Address=%s", namespace, queue, res.GrpcAddress)
		} else {
			logger.ConsoleLog("INFO", "Existing shard returned: Namespace=%s, Queue=%s, Address=%s", namespace, queue, res.GrpcAddress)
		}

		return res.GrpcAddress, res.IsNew, res.NewShardId, nil
	}
	return "", false, 0, nil
}

func (d *ControlStore) getShardManagerConnection() (*grpc.ClientConn, error) {
	if conn, err := grpc.NewClient(d.ShardManagerAddress, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to shard manager at %s: %v", d.ShardManagerAddress, err)
		return conn, fmt.Errorf("failed to connect to shard manager: %v", err)
	} else {
		return conn, nil
	}
}
