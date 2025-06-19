package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/kokaq/protocol/tcp"
)

type PopOperationHandler struct {
	StorageService *StorageService
}

func (handler *PopOperationHandler) Handle(request *tcp.Request) (*tcp.Response, error) {

	handler.StorageService.Logger.Info("Pop operation request, sending success.")
	res := request.ToResponse()
	var queueIdString = request.ToString()
	handler.StorageService.Logger.Info(fmt.Sprintf("Popping queue %v...", queueIdString))
	err := errors.New("could not create queue in storage service")

	heap, ok := handler.StorageService.Heaps[queueIdString]

	if !ok {
		res.SetStatus(tcp.ResponseStatusFail)
		res.SetPayload(make([]byte, 0))
		err = errors.New("Queue does not exists")
		return res, err
	} else {

		responseChannel := make(chan *tcp.Response)
		defer close(responseChannel)
		popRequestToRoutine := &ChannelInput{
			request:         request,
			responseChannel: responseChannel,
		}
		// TODO: Read pop operation timeout from config
		timeout := time.After(time.Second * time.Duration(5))
		handler.StorageService.Logger.Info(fmt.Sprintf("Sending pop request to channel queue %v...", queueIdString))
		heap.channel <- *popRequestToRoutine

		for {
			select {
			case rcv := <-responseChannel:
				handler.StorageService.Logger.Info(fmt.Sprintf("Got a pop response queue %v...", queueIdString))
				return rcv, nil
			case <-timeout:
				res.SetStatus(tcp.ResponseStatusFail)
				res.SetPayload(make([]byte, 0))
				return res, errors.New("Did not receive a response from queue go routine")
			}
		}
	}
}
