package storage

import (
	"github.com/kokaq/protocol/tcp"
)

type NoOperationHandler struct {
	StorageService *StorageService
}

func (handler *NoOperationHandler) Handle(request *tcp.Request) (*tcp.Response, error) {
	handler.StorageService.Logger.Info("No operation request, sending success.")
	res := request.ToResponse()
	res.SetStatus(tcp.ResponseStatusSuccess)
	return res, nil
}
