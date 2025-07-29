package shard

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kokaq/core/utils/murmur"
)

type Shard struct {
	shardId         uint64
	address         string
	internalAddress string
	followers       []string
	updatedAt       time.Time
}

func (s *Shard) GetShardId() uint64 {
	return s.shardId
}
func (s *Shard) GetAddress() string {
	return s.address
}
func (s *Shard) GetInternalAddress() string {
	return s.internalAddress
}
func (s *Shard) GetFollowers() []string {
	return s.followers
}
func (s *Shard) GetUpdatedAt() time.Time {
	return s.updatedAt
}

type DataPlaneShardNode struct {
	Address         string
	InternalAddress string
	LastSeen        time.Time
	IsAlive         bool
}

type ShardStore struct {
	mutex          sync.RWMutex
	nameToShardIds map[string]map[string]uint64
	shards         map[uint32]map[uint32]*Shard
	nodes          map[string]*DataPlaneShardNode
}

func NewShardStore() *ShardStore {
	return &ShardStore{
		mutex:          sync.RWMutex{},
		shards:         make(map[uint32]map[uint32]*Shard, 0),
		nameToShardIds: make(map[string]map[string]uint64, 0),
		nodes:          make(map[string]*DataPlaneShardNode),
	}
}

func (store *ShardStore) RegisterNode(address string, internalAddress string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.nodes[address] = &DataPlaneShardNode{
		Address:         address,
		InternalAddress: internalAddress,
		LastSeen:        time.Now(),
		IsAlive:         true,
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
	for _, ns := range store.shards {
		for _, shard := range ns {
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

	var oldNsId uint32 = 0
	queueMap, nsExists := store.nameToShardIds[namespace]
	if !nsExists {
		queueMap = make(map[string]uint64)
		store.nameToShardIds[namespace] = queueMap
	} else {
		if len(queueMap) > 0 {
			// nsId can be identified
			for _, v := range queueMap {
				nsId, _ := splitShardId(v)
				if nsId != 0 {
					oldNsId = nsId
					break
				}
			}

		}
	}
	_, queueExists := queueMap[queue]
	if queueExists {
		return nil, false, fmt.Errorf("queue already exist for namespace: %s and queue: %s", namespace, queue)
	}

	i := 0
	var shardId uint64
	for {
		i++
		if oldNsId == 0 {
			shardId = generateShardId()
		} else {
			shardId = generateShardIdOfNs(oldNsId)
		}

		nsId, qId := splitShardId(shardId)
		if _, exists := store.shards[nsId][qId]; !exists || i >= 30 {
			break
		}
	}

	if i >= 30 {
		return nil, false, fmt.Errorf("try again")
	}

	address, internalAddress, err := store.allocateShardAddress()
	if err != nil {
		return nil, false, fmt.Errorf("nodes not available for namespace: %s and queue: %s", namespace, queue)
	}
	nsId, qId := splitShardId(shardId)
	store.nameToShardIds[namespace][queue] = shardId
	if _, nsIdExists := store.shards[nsId]; !nsIdExists {
		store.shards[nsId] = make(map[uint32]*Shard)
	}
	store.shards[nsId][qId] = &Shard{
		shardId:         shardId,
		address:         address,
		internalAddress: internalAddress,
		followers:       []string{},
		updatedAt:       time.Now(),
	}

	return store.shards[nsId][qId], true, nil
}

func (store *ShardStore) GetShard(namespace string, queue string) (*Shard, bool) {
	if shardId, exist := store.nameToShardIds[namespace][queue]; !exist {
		return nil, false
	} else {
		nsId, qId := splitShardId(shardId)
		if shard, exist := store.shards[nsId][qId]; !exist {
			return nil, false
		} else {
			return shard, true
		}
	}
}

func (store *ShardStore) GetShards() []*Shard {

	var shards []*Shard = make([]*Shard, len(store.shards))
	i := 0
	for _, ns := range store.shards {
		for _, shard := range ns {
			shards[i] = shard
			i++
		}
	}
	return shards

}

func (store *ShardStore) GetShardById(shardId uint64) (*Shard, bool) {
	nsId, qId := splitShardId(shardId)
	if shard, exist := store.shards[nsId][qId]; !exist {
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
	nsId, qId := splitShardId(shardId)
	delete(store.shards[nsId], qId)
}

func (store *ShardStore) ShardExist(namespace string, queue string) bool {
	if shardId, exist := store.nameToShardIds[namespace][queue]; !exist {
		return false
	} else {
		nsId, qId := splitShardId(shardId)
		if _, exist := store.shards[nsId][qId]; !exist {
			return false
		} else {
			return true
		}
	}
}

func (store *ShardStore) allocateShardAddress() (string, string, error) {
	trueKeys := make([]string, 0)
	for address, node := range store.nodes {
		if node.IsAlive {
			trueKeys = append(trueKeys, address)
		}
	}
	if len(trueKeys) == 0 {
		return "", "", fmt.Errorf("no available address")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	add := trueKeys[r.Intn(len(trueKeys))]
	return add, store.nodes[add].InternalAddress, nil
}

func generateShardId() uint64 {
	rand.Seed(time.Now().UnixNano())
	return (uint64(murmur.SeedNew32(rand.Uint32()).Sum32()) << 32) | uint64(murmur.SeedNew32(rand.Uint32()).Sum32())
}

func generateShardIdOfNs(nsId uint32) uint64 {
	rand.Seed(time.Now().UnixNano())
	return (uint64(nsId) << 32) | uint64(murmur.SeedNew32(rand.Uint32()).Sum32())
}
func splitShardId(shardId uint64) (uint32, uint32) {
	namespaceId := uint32(shardId >> 32)
	queueId := uint32(shardId & 0xFFFFFFFF)
	return namespaceId, queueId
}
