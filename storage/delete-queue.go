package storage

import (
	"errors"
	"fmt"

	"github.com/kokaq/protocol/tcp"
)

type DeleteOperationHandler struct {
	StorageService *StorageService
}

func (handler *DeleteOperationHandler) Handle(request *tcp.Request) (*tcp.Response, error) {
	handler.StorageService.Logger.Info("Delete operation request, sending success.")
	res := request.ToResponse()

	var queueIdString = request.ToString()

	handler.StorageService.Logger.Info(fmt.Sprintf("Deleting queue %v...", queueIdString))

	var err error

	heap, ok := handler.StorageService.Heaps[queueIdString]
	if !ok {
		res.SetStatus(tcp.ResponseStatusFail)
		res.SetPayload(make([]byte, 0))
		err = errors.New("queue does not exist")
	} else {
		handler.StorageService.Logger.Info(fmt.Sprintf("Closing and deleting channel and memory for queue %v...", queueIdString))
		heap.actualHeap.DeleteQueue()
		close(heap.channel)
		delete(handler.StorageService.Heaps, queueIdString)
		handler.StorageService.Logger.Info(fmt.Sprintf("Done deleting queue %v.", queueIdString))
		res.SetStatus(tcp.ResponseStatusSuccess)
		res.SetPayload(make([]byte, 0))
		err = nil
	}
	return res, err
}
