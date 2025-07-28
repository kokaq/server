package data

import (
	"fmt"
	"path/filepath"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/core/queue"
)

type DataStore struct {
	Namespaces       map[uint32]*queue.Namespace
	NamespaceIdIndex map[string]uint32
	ShardIdIndex     map[string]map[string]uint64
}

func NewDataStore() *DataStore {
	return &DataStore{
		Namespaces:       make(map[uint32]*queue.Namespace, 0),
		NamespaceIdIndex: make(map[string]uint32, 0),
		ShardIdIndex:     make(map[string]map[string]uint64, 0),
	}
}

func (store *DataStore) initializeNamespaceIfNotExists(namespaceName string, namespaceId uint32, rootDir string) {
	if _, exists := store.ShardIdIndex[namespaceName]; !exists {
		store.ShardIdIndex[namespaceName] = make(map[string]uint64)
	}

	if nsId, exists := store.NamespaceIdIndex[namespaceName]; exists {
		logger.ConsoleLog("INFO", "Found existing namespace: %s (ID=%x)", namespaceName, nsId)
	} else {
		// Namespace not found
		logger.ConsoleLog("WARN", "Namespace not found: %s. Creating new namespace %x...", namespaceName, namespaceId)
		namespaceDir := filepath.Join(rootDir, fmt.Sprint(namespaceId))
		// Register new namespace
		store.Namespaces[namespaceId] = queue.NewNamespace(namespaceDir, queue.NamespaceConfig{
			NamespaceId:   namespaceId,
			NamespaceName: namespaceName,
		})
		store.NamespaceIdIndex[namespaceName] = namespaceId
	}
}

func (store *DataStore) queueExist(namespaceName string, queueName string) (bool, uint64) {
	if shardId, exists := store.ShardIdIndex[namespaceName][queueName]; exists {
		return true, shardId
	} else {
		return false, shardId
	}
}

func (store *DataStore) createQueue(namespaceName string, queueName string, shardId uint64, enableDeadLetter bool) (bool, uint64, error) {
	namespaceId, queueId := splitShard(shardId)
	if _, err := store.Namespaces[namespaceId].AddQueue(&queue.QueueConfiguration{
		QueueId:   queueId,
		QueueName: queueName,
		EnableDLQ: enableDeadLetter,
	}); err != nil {
		return false, shardId, fmt.Errorf("failed to add new queue:%v", err)
	}
	store.ShardIdIndex[namespaceName][queueName] = shardId
	return true, shardId, nil
}

func (store *DataStore) deleteQueue(namespaceName string, queueName string) (bool, error) {
	shardId, exist := store.ShardIdIndex[namespaceName][queueName]
	namespaceId, queueId := splitShard(shardId)
	if exist {
		delete(store.ShardIdIndex[namespaceName], queueName)
		store.Namespaces[namespaceId].DeleteQueue(queueId)
	}
	return true, nil
}

func (store *DataStore) clearQueue(namespaceName string, queueName string) (bool, error) {
	shardId, exist := store.ShardIdIndex[namespaceName][queueName]
	namespaceId, queueId := splitShard(shardId)
	if exist {
		store.Namespaces[namespaceId].ClearQueue(queueId)
	}
	return true, nil
}

func (store *DataStore) getQueue(namespace string, queue string) (*queue.Queue, error) {
	shardId, exists := store.ShardIdIndex[namespace][queue]
	if !exists {
		return nil, fmt.Errorf("queue does not exist")
	}
	namespaceId, queueId := splitShard(shardId)
	ns, exists := store.Namespaces[namespaceId]
	if !exists {
		return nil, fmt.Errorf("namespace does not exist")
	}
	var err error = nil
	if q, err := ns.GetQueue(queueId); err == nil {
		return q, nil
	}
	return nil, err
}

func splitShard(shardId uint64) (uint32, uint32) {
	namespaceId := uint32(shardId >> 32)
	queueId := uint32(shardId & 0xFFFFFFFF)
	return namespaceId, queueId
}
