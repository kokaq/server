package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/core/queue"
	"github.com/kokaq/protocol/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DataPlane struct {
	proto.UnimplementedKokaqDataPlaneServer
	RootDir string
	store   *DataStore
}

func NewDataPlane(rootDirectory string) (*DataPlane, error) {
	return &DataPlane{
		RootDir: rootDirectory,
		store:   NewDataStore(),
	}, nil
}

func (d *DataPlane) New(c context.Context, p *proto.KokaqNewQueueRequest) (*proto.KokaqQueueResponse, error) {
	logger.ConsoleLog("INFO", "Received new queue request: Namespace=%s, Queue=%s, ShardId=%x", p.Request.Namespace, p.Request.Queue, p.ShardId)

	d.store.initializeNamespaceIfNotExists(p.Request.Namespace, uint32(p.ShardId>>32), d.RootDir)

	// Try to get existing queue ID
	exists, _ := d.store.queueExist(p.Request.Namespace, p.Request.Queue)
	if !exists {
		// Queue not found — create a new one using lower 32 bits of shard ID
		logger.ConsoleLog("WARN", "Queue not found: %s. Creating new queue...", p.Request.Queue)
		queueId := uint32(p.ShardId & 0xFFFFFFFF)
		_, _, err := d.store.createQueue(p.Request.Namespace, p.Request.Queue, p.ShardId, false)
		if err != nil {
			logger.ConsoleLog("ERROR", "%v", err)
			return &proto.KokaqQueueResponse{ShardId: p.ShardId}, err
		}
		logger.ConsoleLog("INFO", "Successfully created queue: %s (ID=%x)", p.Request.Queue, queueId)
		return &proto.KokaqQueueResponse{
			ShardId:        p.ShardId,
			TotalNodeCount: 0,
			TotalPageCount: 0,
			CreatedOn:      timestamppb.Now(),
			Request:        p.Request,
		}, nil
	}
	// Queue already exists
	logger.ConsoleLog("ERROR", "Queue already exists: %s", p.Request.Queue)
	return &proto.KokaqQueueResponse{}, fmt.Errorf("queue already exists")
}

func (d *DataPlane) Get(c context.Context, p *proto.KokaqQueueRequest) (*proto.KokaqQueueResponse, error) {
	logger.ConsoleLog("INFO", "Received queue lookup request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	var err error
	var res *proto.KokaqQueueResponse
	exists, shardId := d.store.queueExist(p.Namespace, p.Queue)
	if exists {
		logger.ConsoleLog("INFO", "Successfully resolved queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
		res = &proto.KokaqQueueResponse{
			TotalNodeCount: 0,
			TotalPageCount: 0,
			CreatedOn:      timestamppb.Now(),
			Request:        p,
			ShardId:        shardId,
		}
		err = nil
	} else {
		logger.ConsoleLog("ERROR", "queue %s not found in namespace %s", p.Queue, p.Namespace)
		res, err = nil, fmt.Errorf("queue %s not found in namespace %s", p.Queue, p.Namespace)
	}
	return res, err
}

func (d *DataPlane) GetStats(c context.Context, p *proto.KokaqQueueRequest) (*proto.KokaqStatsResponse, error) {
	return &proto.KokaqStatsResponse{Stats: nil, Status: &proto.StatusResponse{}}, fmt.Errorf("cannot get stats")
}

func (d *DataPlane) Delete(c context.Context, p *proto.KokaqQueueRequest) (*proto.StatusResponse, error) {
	logger.ConsoleLog("INFO", "Received queue delete request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	deleted, err := d.store.deleteQueue(p.Namespace, p.Queue)
	if !deleted {
		logger.ConsoleLog("ERROR", "Delete - failed to delete: %v", err)
		return &proto.StatusResponse{}, err
	}
	logger.ConsoleLog("INFO", "Successfully deleted queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	return &proto.StatusResponse{}, nil
}

func (d *DataPlane) Clear(c context.Context, p *proto.KokaqQueueRequest) (*proto.StatusResponse, error) {
	logger.ConsoleLog("INFO", "Received queue clear request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	deleted, err := d.store.clearQueue(p.Namespace, p.Queue)
	if !deleted {
		logger.ConsoleLog("ERROR", "CLEAR - failed to clear: %v", err)
		return &proto.StatusResponse{}, err
	}
	logger.ConsoleLog("INFO", "Successfully cleared queue: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	return &proto.StatusResponse{}, nil
}

func (d *DataPlane) Enqueue(c context.Context, p *proto.EnqueueRequest) (*proto.EnqueueResponse, error) {
	logger.ConsoleLog("INFO", "Received enqueue request: Namespace=%s, Queue=%s", p.Message.Namespace, p.Message.Queue)
	q, err := d.store.getQueue(p.Message.Namespace, p.Message.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Enqueue - queue not found: %v", err)
		return &proto.EnqueueResponse{}, err
	}
	mId := uuid.New()
	if err := q.Enqueue(&queue.QueueItem{MessageId: mId, Priority: p.Message.Priority}); err != nil {
		logger.ConsoleLog("ERROR", "Enqueue - failed to enqueue: %v", err)
		return &proto.EnqueueResponse{}, err
	}
	return &proto.EnqueueResponse{}, nil
}

func (d *DataPlane) Dequeue(c context.Context, p *proto.DequeueRequest) (*proto.DequeueResponse, error) {
	logger.ConsoleLog("INFO", "Received dequeue request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Dequeue - queue not found: %v", err)
		return &proto.DequeueResponse{}, err
	}
	qi, err := q.Dequeue()
	if err != nil {
		logger.ConsoleLog("ERROR", "Dequeue - failed: %v", err)
		return &proto.DequeueResponse{}, err
	}
	var messages = make([]*proto.KokaqMessageResponse, 0)
	var message = &proto.KokaqMessageResponse{
		Message: &proto.KokaqMessageRequest{
			MessageId: qi.MessageId.String(),
			Priority:  qi.Priority,
		},
	}
	messages = append(messages, message)
	return &proto.DequeueResponse{Messages: messages}, nil
}

func (d *DataPlane) Peek(c context.Context, p *proto.PeekRequest) (*proto.PeekResponse, error) {
	logger.ConsoleLog("INFO", "Received peek request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Peek - queue not found: %v", err)
		return &proto.PeekResponse{}, err
	}
	qi, err := q.Peek()
	if err != nil {
		logger.ConsoleLog("ERROR", "Peek - failed: %v", err)
		return &proto.PeekResponse{}, err
	}
	var messages = make([]*proto.KokaqMessageResponse, 0)
	var message = &proto.KokaqMessageResponse{
		Message: &proto.KokaqMessageRequest{
			MessageId: qi.MessageId.String(),
			Priority:  qi.Priority,
		},
	}
	messages = append(messages, message)
	return &proto.PeekResponse{Messages: messages}, nil
}

func (d *DataPlane) PeekLock(c context.Context, p *proto.PeekLockRequest) (*proto.PeekLockResponse, error) {
	logger.ConsoleLog("INFO", "Received peeklock request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "PeekLock - queue not found: %v", err)
		return &proto.PeekLockResponse{}, err
	}
	qi, lockId, err := q.PeekLock()
	if err != nil {
		logger.ConsoleLog("ERROR", "PeekLock - failed: %v", err)
		return &proto.PeekLockResponse{}, err
	}
	var messages = make([]*proto.LockedMessage, 0)
	var message = &proto.LockedMessage{
		Message: &proto.KokaqMessageResponse{
			Message: &proto.KokaqMessageRequest{
				MessageId: qi.MessageId.String(),
				Priority:  qi.Priority,
			},
		},
		LockId: lockId,
	}
	messages = append(messages, message)
	return &proto.PeekLockResponse{Locked: messages}, nil
}

func (d *DataPlane) Ack(c context.Context, p *proto.AckRequest) (*proto.AckResponse, error) {
	logger.ConsoleLog("INFO", "Received ack request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Ack - queue not found: %v", err)
		return &proto.AckResponse{Acknowledged: false}, err
	}
	if err := q.Ack(p.LockId); err != nil {
		logger.ConsoleLog("ERROR", "Ack - failed: %v", err)
		return &proto.AckResponse{Acknowledged: false}, err
	}
	return &proto.AckResponse{Acknowledged: true}, nil
}

func (d *DataPlane) Nack(c context.Context, p *proto.NackRequest) (*proto.NackResponse, error) {
	logger.ConsoleLog("INFO", "Received nack request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Nack - queue not found: %v", err)
		return &proto.NackResponse{}, err
	}
	if err := q.Nack(p.LockId); err != nil {
		logger.ConsoleLog("ERROR", "Nack - failed: %v", err)
		return &proto.NackResponse{}, err
	}
	return &proto.NackResponse{Requeued: true}, nil
}

func (d *DataPlane) Extend(c context.Context, p *proto.ExtendVisibilityTimeoutRequest) (*proto.VisibilityTimeoutResponse, error) {
	logger.ConsoleLog("INFO", "Received extend request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "Extend - queue not found: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	if err := q.Extend(p.LockId, time.Duration(p.AdditionalMs)*time.Millisecond); err != nil {
		logger.ConsoleLog("ERROR", "Extend - failed: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	return &proto.VisibilityTimeoutResponse{Applied: true}, nil
}

func (d *DataPlane) SetVisibilityTimeout(c context.Context, p *proto.SetVisibilityTimeoutRequest) (*proto.VisibilityTimeoutResponse, error) {
	logger.ConsoleLog("INFO", "Received SetVisibilityTimeout request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "SetVisibilityTimeout - queue not found: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	if err := q.SetVisibilityTimeout(time.Duration(p.NewTimeoutMs) * time.Millisecond); err != nil {
		logger.ConsoleLog("ERROR", "SetVisibilityTimeout - failed: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	return &proto.VisibilityTimeoutResponse{Applied: true}, nil
}

func (d *DataPlane) RefreshVisibilityTimeout(c context.Context, p *proto.RefreshVisibilityTimeoutRequest) (*proto.VisibilityTimeoutResponse, error) {
	logger.ConsoleLog("INFO", "Received RefreshVisibilityTimeout request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "RefreshVisibilityTimeout - queue not found: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	if err := q.RefreshVisibilityTimeout(p.LockId); err != nil {
		logger.ConsoleLog("ERROR", "RefreshVisibilityTimeout - failed: %v", err)
		return &proto.VisibilityTimeoutResponse{Applied: false}, err
	}
	return &proto.VisibilityTimeoutResponse{Applied: true}, nil
}

func (d *DataPlane) ReleaseLock(c context.Context, p *proto.ReleaseLockRequest) (*proto.ReleaseLockResponse, error) {
	logger.ConsoleLog("INFO", "Received ReleaseLock request: Namespace=%s, Queue=%s", p.Namespace, p.Queue)
	q, err := d.store.getQueue(p.Namespace, p.Queue)
	if err != nil {
		logger.ConsoleLog("ERROR", "ReleaseLock - queue not found: %v", err)
		return &proto.ReleaseLockResponse{Released: false}, err
	}
	if err := q.ReleaseLock(p.LockId); err != nil {
		logger.ConsoleLog("ERROR", "ReleaseLock - failed: %v", err)
		return &proto.ReleaseLockResponse{Released: false}, err
	}
	return &proto.ReleaseLockResponse{Released: true}, nil
}

// func (d *DataPlane) IsExpired(c context.Context, p *proto.LockIdRequest) (*proto.IsExpiredResponse, error) {
// 	var err error
// 	if q, err := d.forQueue(p.NamespaceId, p.QueueId); err == nil {
// 		if expired, err := q.IsExpired(p.LockId); err == nil {
// 			return &proto.IsExpiredResponse{IsExpired: expired, Error: nil}, nil
// 		}
// 	}
// 	return &proto.IsExpiredResponse{Error: &proto.Error{Message: d.errString(err), Code: 0}, IsExpired: false}, err
// }

// func (d *DataPlane) GetLockedMessages(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemsResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) MoveToDLQ(c context.Context, p *proto.MoveToDLQRequest) (*proto.EmptyResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) AutoMoveToDLQ(c context.Context, p *proto.AutoMoveToDLQRequest) (*proto.EmptyResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) PeekDLQ(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemsResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) DequeueDLQ(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) MoveFromDLQ(c context.Context, p *proto.MoveToDLQRequest) (*proto.EmptyResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) ClearDLQ(c context.Context, p *proto.QueueIdRequest) (*proto.EmptyResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) ListMessages(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemsResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) ListLockedMessages(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemsResponse, error) {
// 	panic("unimplemented")
// }

// func (d *DataPlane) ListDLQMessages(c context.Context, p *proto.QueueIdRequest) (*proto.QueueItemsResponse, error) {
// 	panic("unimplemented")
// }
