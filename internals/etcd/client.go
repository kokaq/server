package etcd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdClient struct {
	logger  *logrus.Logger
	Client  *clientv3.Client
	timeout time.Duration
}

func NewEtcdClient(etcdEndpointsString string, etcdTimeoutInSeconds int, logger *logrus.Logger) (*EtcdClient, error) {
	logger.Info("Creating etcd client...")

	etcdEndpoints := strings.Split(etcdEndpointsString, ",")

	cfg := clientv3.Config{
		Endpoints: etcdEndpoints,
	}

	c, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Created etcd client.")

	return &EtcdClient{
		logger:  logger,
		Client:  c,
		timeout: time.Second * time.Duration(etcdTimeoutInSeconds),
	}, nil
}

func (etcd *EtcdClient) Get(key string) (string, error) {
	etcd.logger.Info(fmt.Sprintf("Getting key %v...", key))

	ctx, cancel := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancel()
	response, err := etcd.Client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	numResponses := len(response.Kvs)
	if numResponses == 0 {
		return "", fmt.Errorf("key not found [%v]", key)
	}

	etcd.logger.Info(fmt.Sprintf("Got key %v.", key))
	return string(response.Kvs[0].Value), nil
}

func (etcd *EtcdClient) Put(key string, value string) error {
	etcd.logger.Info(fmt.Sprintf("Putting key %v value %v...", key, value))

	ctx, cancel := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancel()

	_, err := etcd.Client.Put(ctx, key, value)

	etcd.logger.Info(fmt.Sprintf("Done putting key %v value %v.", key, value))

	return err
}

func (etcd *EtcdClient) Delete(key string) error {
	etcd.logger.Info(fmt.Sprintf("Deleting key %v...", key))

	ctx, cancel := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancel()

	_, err := etcd.Client.Delete(ctx, key)

	etcd.logger.Info(fmt.Sprintf("Deleted key %v.", key))

	return err
}

func (etcd *EtcdClient) DeleteWithPrefix(key string) error {
	etcd.logger.Info(fmt.Sprintf("Deleting key prefix %v...", key))

	ctx, cancel := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancel()

	_, err := etcd.Client.Delete(ctx, key, clientv3.WithPrefix())

	etcd.logger.Info(fmt.Sprintf("Deleted key prefix %v.", key))

	return err
}
