package shard

import (
	"context"
	"fmt"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/protocol/proto"
)

type ShardPlane struct {
	proto.UnimplementedKokaqShardManagerServer
	store *ShardStore
}

func NewShardPlane(rootDirectory string) (*ShardPlane, error) {
	return &ShardPlane{
		store: NewShardStore(),
	}, nil
}

func (d *ShardPlane) RegisterNode(c context.Context, p *proto.RegisterNodeRequest) (*proto.RegisterNodeResponse, error) {
	logger.ConsoleLog("INFO", "Registering shard at address: %s", p.GrpcAddress)
	d.store.RegisterNode(p.GrpcAddress, p.InternalAddress)
	logger.ConsoleLog("INFO", "Shard registration successful for address: %s", p.GrpcAddress)
	return &proto.RegisterNodeResponse{Accepted: true}, nil
}

func (d *ShardPlane) UnregisterNode(c context.Context, p *proto.RegisterNodeRequest) (*proto.RegisterNodeResponse, error) {
	logger.ConsoleLog("INFO", "Unregistering shard at address: %s", p.GrpcAddress)
	d.store.UnregisterNode(p.GrpcAddress)
	logger.ConsoleLog("INFO", "Shard unregistration complete for address: %s", p.GrpcAddress)
	return &proto.RegisterNodeResponse{Accepted: true}, nil
}

func (s *ShardPlane) GetShard(ctx context.Context, p *proto.GetShardRequest) (*proto.GetShardResponse, error) {
	// Combine NamespaceId and QueueId to form a unique shardId
	shardId := (uint64(p.NamespaceId) << 32) | uint64(p.QueueId)
	sh, found := s.store.GetShardById(shardId)
	if found {
		logger.ConsoleLog("DEBUG", "Found existing shardId=%x with address=%s", sh.GetShardId(), sh.GetAddress())
		return &proto.GetShardResponse{
			GrpcAddress:     sh.GetAddress(),
			InternalAddress: sh.GetInternalAddress(),
			IsNew:           false,
		}, nil
	} else {
		sh, found = s.store.GetShard(p.Namespace, p.Queue)
		if found {
			logger.ConsoleLog("DEBUG", "Found existing shardId=%x with address=%s", sh.GetShardId(), sh.GetAddress())
			return &proto.GetShardResponse{
				GrpcAddress:     sh.GetAddress(),
				InternalAddress: sh.GetInternalAddress(),
				IsNew:           false,
			}, nil
		}
	}

	if !p.CreateIfNotFound {
		logger.ConsoleLog("WARN", "Shard not found and CreateIfNotFound is false for shardId=%x", shardId)
		return &proto.GetShardResponse{
			Status: &proto.StatusResponse{
				Success: false,
				Error:   proto.ErrorCode_ERROR_NOT_FOUND,
			},
		}, fmt.Errorf("shard does not exist")
	}

	// Find an available shard address
	shrd, _, err := s.store.AllocateShard(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Could not allocate shard to queue: %v", err)
		return &proto.GetShardResponse{
			GrpcAddress:     "",
			InternalAddress: "",
			Status: &proto.StatusResponse{
				Success: false,
				Error:   proto.ErrorCode_ERROR_NOT_FOUND,
			},
		}, err
	}
	logger.ConsoleLog("INFO", "Allocated new shardId=%x to address=%s", shrd.GetShardId(), shrd.GetAddress())
	return &proto.GetShardResponse{
		GrpcAddress:     shrd.GetAddress(),
		InternalAddress: shrd.GetInternalAddress(),
		IsNew:           true,
		NewShardId:      shrd.shardId,
	}, nil
}

func (s *ShardPlane) DeleteShard(ctx context.Context, p *proto.GetShardRequest) (*proto.StatusResponse, error) {
	sh, found := s.store.GetShard(p.Namespace, p.Queue)
	if found {
		logger.ConsoleLog("DEBUG", "Found existing shardId=%x with address=%s", sh.GetShardId(), sh.GetAddress())
		s.store.DeleteShard(p.Namespace, p.Queue)
		return &proto.StatusResponse{
			Success: true,
		}, nil
	}
	return &proto.StatusResponse{
		Success: false,
	}, fmt.Errorf("shard for this queue does not exist")
}

func (s *ShardPlane) RequestShard(ctx context.Context, p *proto.GetShardRequest) (*proto.GetShardResponse, error) {
	// Check if shard already assigned for this namespace and queue
	shrd, _, err := s.store.AllocateShard(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("WARN", "Cannot allocate shard for namespace=%s, queue=%s, %v", p.Namespace, p.Queue, err)
		err = fmt.Errorf("cannot allocate shard for namespace=%s, queue=%s, %v", p.Namespace, p.Queue, err)
		return &proto.GetShardResponse{
			Status: &proto.StatusResponse{
				Success: false,
				Error:   proto.ErrorCode_ERROR_QUEUE_DISABLED,
			},
		}, err
	}
	logger.ConsoleLog("INFO", "Allocated shardId=%x to address=%s for namespace=%s, queue=%s",
		shrd.GetShardId(), shrd.GetAddress(), p.Namespace, p.Queue)
	return &proto.GetShardResponse{
		GrpcAddress: shrd.GetAddress(),
		IsNew:       true,
		NewShardId:  shrd.GetShardId(),
	}, nil
}

func (s *ShardPlane) ListShards(ctx context.Context, p *proto.ListShardsRequest) (*proto.ListShardsResponse, error) {
	var shards []*proto.ShardItem
	// for shardId, addr := range s.Shards {
	// 	if s.AvailableAddrs[addr] {
	// 		namespaceId := uint32(shardId >> 32)
	// 		// queueId  := uint32(shardId & 0xFFFFFFFF)
	// 		shards = append(shards, &proto.ShardItem{
	// 			NamespaceId: namespaceId,
	// 			QueueIds: make([]uint32, 0),
	// 			LastCheckin: uint64(time.Now().UTC().Unix()),
	// 			GrpcAddress: addr,
	// 		})
	// 	}
	// }

	return &proto.ListShardsResponse{
		Status: &proto.StatusResponse{Success: false, Error: proto.ErrorCode_ERROR_DEPENDENCY_FAILURE},
		Shards: shards,
	}, nil
}
