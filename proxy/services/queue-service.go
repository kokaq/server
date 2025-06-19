package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	client "github.com/kokaq/client/tcp"
	"github.com/kokaq/protocol/tcp"
	"github.com/kokaq/server/internals/etcd"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type Queue struct {
	// All properties with json schema
	Name        string
	Description string
	MaxSize     int
}

type QueueServiceConfig struct {
	MessageContentEtcdEndpoints    string
	MessageContentDbEtcdTimeout    int
	PodMappingDbEtcdEndpoints      string
	PodMappingDbEtcdTimeout        int
	StorageServiceTcpTimeout       int
	StorageServicePodsFetchTimeout int
	StorageServiceTcpRetries       int
}

type QueueService struct {
	logger                         *logrus.Logger
	MessageContentDb               *etcd.EtcdClient
	PodMappingDb                   *etcd.EtcdClient
	StorageServiceTcpTimeout       int
	StorageServicePodsFetchTimeout int
	StorageServiceTcpRetries       int
	InMemoryCache                  *cache.Cache
	StorageServicePods             []string
	TotalStorageServicePods        int
}

func NewQueueService(logger *logrus.Logger, config QueueServiceConfig) (*QueueService, error) {
	var mesEtcdClinet, err1 = etcd.NewEtcdClient(config.MessageContentEtcdEndpoints, config.MessageContentDbEtcdTimeout, logger)
	var podMappingDb, err2 = etcd.NewEtcdClient(config.PodMappingDbEtcdEndpoints, config.PodMappingDbEtcdTimeout, logger)

	if err1 != nil {
		return nil, err1
	}

	if err2 != nil {
		return nil, err2
	}

	return &QueueService{
		logger:                         logger,
		MessageContentDb:               mesEtcdClinet,
		PodMappingDb:                   podMappingDb,
		StorageServiceTcpTimeout:       config.StorageServiceTcpTimeout,
		StorageServicePodsFetchTimeout: config.StorageServicePodsFetchTimeout,
		StorageServiceTcpRetries:       config.StorageServiceTcpRetries,
		InMemoryCache:                  cache.New(60*time.Minute, 60*time.Minute),
	}, nil
}

func NewQueueServiceFromContext(ctx context.Context) *QueueService {
	if val := ctx.Value("queue-service"); val != nil {
		return val.(*QueueService)
	}
	return nil
}

func (qs *QueueService) CreateQueue(queue Queue) *Queue {
	return &queue
}

func (qs *QueueService) DeleteQueue(id string) {
}

func (qs *QueueService) GetQueue(id string) *Queue {
	return &Queue{}
}

func (qs *QueueService) ListQueues() []*Queue {
	return []*Queue{
		{},
		{},
	}
}

func (qs *QueueService) UpdateQueue(id string, queue Queue) *Queue {
	return &queue
}

func (qs *QueueService) SendStorageServiceRequest(storageServiceAddress string, request *tcp.Request) (*tcp.Response, error) {
	qs.logger.Info(fmt.Sprintf("Sending storage service request to address %v", storageServiceAddress))

	err := errors.New("could not reach storage service pod")

	for i := 0; i < qs.StorageServiceTcpRetries; i++ {
		if err != nil {
			qs.logger.Error(fmt.Sprintf("An error occured during a retry for send storage service request %v", err.Error()))
		}

		qs.logger.Info(fmt.Sprintf(("Trying to send storage service request to address %v, try number %v"), storageServiceAddress, i+1))

		// Making tcp request
		var client = client.NewTcpClientFromAddress(storageServiceAddress, qs.StorageServiceTcpTimeout)
		var resp, err2 = client.Send(request)
		if err2 != nil {
			qs.logger.Error(fmt.Sprintf("storage service did not respond %v", err.Error()))
			err = err2
			continue
		}

		qs.logger.Info(fmt.Sprintf("Sent storage service request to address %v", storageServiceAddress))

		qs.logger.Info(fmt.Sprintf("Done sending request to storage service address %v", storageServiceAddress))

		// might be success and no mismatch
		// might be failed
		return resp, nil
	}

	return nil, err
}

func (qs *QueueService) GetRandomStorageServicePod() (string, error) {
	qs.logger.Info("Getting random storage service pod...")

	// Check if cache of pods needs to be refreshed in case of timeout (scaling scenarios)
	if _, found := qs.InMemoryCache.Get("QUEUE_SERVICE_STORAGE_SERVICE_PODS_CACHE_KEY"); !found {
		qs.logger.Info("Setting storage service pods from kubernetes sdk...")

		err := qs.SetStorageServicePodsFromKubernetes()
		if err != nil {
			return "", err
		}

		qs.InMemoryCache.Set("QUEUE_SERVICE_STORAGE_SERVICE_PODS_CACHE_KEY",
			"QUEUE_SERVICE_STORAGE_SERVICE_PODS_CACHE_KEY", time.Second*time.Duration(qs.StorageServicePodsFetchTimeout))

		qs.logger.Info("Done settings storage service pods from kubernetes.")
	}

	// Get random pod from storage service pods
	randomIndex := rand.Intn(qs.TotalStorageServicePods)

	return qs.StorageServicePods[randomIndex], nil
}

func (qs *QueueService) SetStorageServicePodsFromKubernetes() error {
	qs.logger.Info("Getting storage service pods from kubernetes...")

	// Set storage service pods
	qs.StorageServicePods = make([]string, 0)
	qs.TotalStorageServicePods = 0

	qs.logger.Info(fmt.Sprintf("Done getting storage service pods [total count: %v] from kubernetes.", qs.TotalStorageServicePods))

	return nil
}

func (qs *QueueService) GetStorageServiceAddressForQueue(queueId string) (string, error) {
	qs.logger.Info(fmt.Sprintf("Getting storage service address for queue %v", queueId))

	// Get storage service address from pod mapping db
	storageServiceAddress, err := qs.PodMappingDb.Get(queueId)
	if err != nil {
		return "", nil
	}

	qs.logger.Info(fmt.Sprintf("Done getting storage service address for queue %v", queueId))

	return storageServiceAddress, nil
}

func (qs *QueueService) SetStorageServiceAddressForQueue(queueId string, storageServiceAddress string) error {

	qs.logger.Info(fmt.Sprintf("Setting storage service address for queue %v", queueId))

	// Set storage service address from pod mapping db
	err := qs.PodMappingDb.Put(queueId, storageServiceAddress)
	if err != nil {
		return err
	}

	qs.logger.Info(fmt.Sprintf("Done setting storage service address for queue %v", queueId))

	return nil
}

func (qs *QueueService) GetMessageContentForMessage(messageId string) (string, error) {
	return qs.MessageContentDb.Get(messageId)
}

func (qs *QueueService) SetMessageContentForMessage(messageId string, messageContent string) error {
	return qs.MessageContentDb.Put(messageId, messageContent)
}
