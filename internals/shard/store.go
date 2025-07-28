package shard

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kokaq/core/utils/murmur"
)

type Shard struct {
	shardId   uint64
	address   string
	followers []string
	updatedAt time.Time
}

func (s *Shard) GetShardId() uint64 {
	return s.shardId
}
func (s *Shard) GetAddress() string {
	return s.address
}
func (s *Shard) GetFollowers() []string {
	return s.followers
}
func (s *Shard) GetUpdatedAt() time.Time {
	return s.updatedAt
}

type DataPlaneShardNode struct {
	Address  string
	LastSeen time.Time
	IsAlive  bool
}

type ShardStore struct {
	mutex          sync.RWMutex
	nameToShardIds map[string]map[string]uint64
	shards         map[uint64]*Shard
	nodes          map[string]*DataPlaneShardNode
}

func NewShardStore() *ShardStore {
	return &ShardStore{
		mutex:          sync.RWMutex{},
		shards:         make(map[uint64]*Shard, 0),
		nameToShardIds: make(map[string]map[string]uint64, 0),
		nodes:          make(map[string]*DataPlaneShardNode),
	}
}

func (store *ShardStore) RegisterNode(address string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.nodes[address] = &DataPlaneShardNode{
		Address:  address,
		LastSeen: time.Now(),
		IsAlive:  true,
	}

	return nil
}

func (store *ShardStore) UnregisterNode(address string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, exist := store.nodes[address]; !exist {
		return fmt.Errorf("node not found")
	}
	delete(store.nodes, address)
	// Optionally: remove this node from leader/follower lists of shards
	for _, shard := range store.shards {
		if shard.address == address {
			shard.address = ""
		}
		var updatedFollowers []string
		for _, f := range shard.followers {
			if f != address {
				updatedFollowers = append(updatedFollowers, f)
			}
		}
		shard.followers = updatedFollowers
	}
	return nil
}

func (store *ShardStore) Heartbeat(address string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if node, exist := store.nodes[address]; exist {
		node.LastSeen = time.Now()
		node.IsAlive = true
		return nil
	} else {
		store.nodes[address] = &DataPlaneShardNode{
			Address:  address,
			LastSeen: time.Now(),
			IsAlive:  true,
		}
	}
	return nil
}

func (store *ShardStore) NodeMonitor() {
	time.Sleep(5 * time.Second)
	store.mutex.Lock()
	now := time.Now()
	for _, node := range store.nodes {
		if now.Sub(node.LastSeen) > 10*time.Second {
			node.IsAlive = false
		}
	}
	store.mutex.Unlock()
}

func (store *ShardStore) AllocateShard(namespace string, queue string) (shrd *Shard, allocated bool, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	queueMap, nsExists := store.nameToShardIds[namespace]
	if !nsExists {
		queueMap = make(map[string]uint64)
		store.nameToShardIds[namespace] = queueMap
	}
	_, queueExists := queueMap[queue]
	if queueExists {
		return nil, false, fmt.Errorf("queue already exist for namespace: %s and queue: %s", namespace, queue)
	}

	i := 0
	var shardId uint64
	for {
		i++
		shardId = generateShardId()
		queueMap[queue] = shardId

		if _, exists := store.shards[shardId]; !exists || i >= 30 {
			break
		}
	}

	if i >= 30 {
		return nil, false, fmt.Errorf("try again")
	}

	address, err := store.allocateShardAddress()
	if err != nil {
		return nil, false, fmt.Errorf("nodes not available for namespace: %s and queue: %s", namespace, queue)
	}

	store.shards[shardId] = &Shard{
		shardId:   shardId,
		address:   address,
		followers: []string{},
		updatedAt: time.Now(),
	}

	return store.shards[shardId], true, nil
}

func (store *ShardStore) GetShard(namespace string, queue string) (*Shard, bool) {
	if shardId, exist := store.nameToShardIds[namespace][queue]; !exist {
		return nil, false
	} else {
		if shard, exist := store.shards[shardId]; !exist {
			return nil, false
		} else {
			return shard, true
		}
	}
}

func (store *ShardStore) GetShards() []*Shard {

	var shards []*Shard = make([]*Shard, len(store.shards))
	i := 0
	for _, shard := range store.shards {
		shards[i] = shard
		i++
	}
	return shards

}

func (store *ShardStore) GetShardById(shardId uint64) (*Shard, bool) {
	if shard, exist := store.shards[shardId]; !exist {
		return nil, false
	} else {
		return shard, true
	}
}

func (store *ShardStore) DeleteShard(namespace string, queue string) {
	shardId, exist := store.nameToShardIds[namespace][queue]
	if exist {
		delete(store.nameToShardIds[namespace], queue)
	}
	delete(store.shards, shardId)
}

func (store *ShardStore) ShardExist(namespace string, queue string) bool {
	if shardId, exist := store.nameToShardIds[namespace][queue]; !exist {
		return false
	} else {
		if _, exist := store.shards[shardId]; !exist {
			return false
		} else {
			return true
		}
	}
}

func (store *ShardStore) allocateShardAddress() (string, error) {
	trueKeys := make([]string, 0)
	for address, node := range store.nodes {
		if node.IsAlive {
			trueKeys = append(trueKeys, address)
		}
	}
	if len(trueKeys) == 0 {
		return "", fmt.Errorf("no available address")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return trueKeys[r.Intn(len(trueKeys))], nil
}

func generateShardId() uint64 {
	rand.Seed(time.Now().UnixNano())
	return (uint64(murmur.SeedNew32(rand.Uint32()).Sum32()) << 32) | uint64(murmur.SeedNew32(rand.Uint32()).Sum32())
}
