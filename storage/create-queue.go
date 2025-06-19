package storage

import (
	"errors"
	"fmt"

	"github.com/kokaq/protocol/tcp"
)

type CreateOperationHandler struct {
	StorageService *StorageService
}

func (handler *CreateOperationHandler) Handle(request *tcp.Request) (*tcp.Response, error) {
	handler.StorageService.Logger.Info("Create operation request, sending success.")
	res := request.ToResponse()
	var queueIdString = request.ToString()
	handler.StorageService.Logger.Info(fmt.Sprintf("Creating queue %v...", queueIdString))

	var err error

	_, ok := handler.StorageService.Heaps[queueIdString]
	if ok {
		res.SetStatus(tcp.ResponseStatusFail)
		res.SetPayload(make([]byte, 0))
		err = errors.New("Queue already exists")
	} else {
		// Make channel to interact with queue
		mainChannel := make(chan ChannelInput)
		heap := NewHeap(mainChannel, request.GetNamespaceId(), request.GetQueueId())

		handler.StorageService.Heaps[queueIdString] = heap

		go heap.handle()

		handler.StorageService.Logger.Info(fmt.Sprintf("Done creating queue %v.", queueIdString))
		res.SetStatus(tcp.ResponseStatusSuccess)
		res.SetPayload([]byte(queueIdString))
		err = nil
	}
	return res, err
}
