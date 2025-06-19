package storage

import (
	"fmt"
	"time"

	"github.com/kokaq/protocol/tcp"
	tcp_server "github.com/kokaq/server/tcp"
	"github.com/sirupsen/logrus"
)

type StorageService struct {
	Logger   *logrus.Logger
	Heaps    map[string]*Heap
	Handlers map[int]tcp.Handler
}

func NewStorageService(port int, logger *logrus.Logger) (*StorageService, *tcp_server.Server, error) {
	// Create instance
	ss := &StorageService{}
	server := tcp_server.NewServer(tcp_server.ServerConfig{
		Port:           port,
		Timeout:        30,
		UseMTls:        false,
		MaxConnections: 1000,
		CertFile:       "",
		KeyFile:        "",
	})

	// Initialize logger
	ss.Logger = logger
	ss.Logger.Info("Creating storage service...")

	ss.Heaps = make(map[string]*Heap)
	// Map handlers for storage service
	ss.Handlers = map[int]tcp.Handler{
		0: &NoOperationHandler{
			StorageService: ss,
		},
		1: &CreateOperationHandler{
			StorageService: ss,
		},
		2: &DeleteOperationHandler{
			StorageService: ss,
		},
		3: &NoOperationHandler{
			StorageService: ss,
		},
		4: &PeekOperationHandler{
			StorageService: ss,
		},
		5: &PopOperationHandler{
			StorageService: ss,
		},
		6: &PushOperationHandler{
			StorageService: ss,
		},
		7: &NoOperationHandler{ // Peek lock queue handler
			StorageService: ss,
		},
		8: &NoOperationHandler{ // Release lock queue handler
			StorageService: ss,
		},
	}

	ss.Logger.Info("Created storage service.")

	return ss, server, nil
}

func (qs *StorageService) HandleRequest(req *tcp.Request) (*tcp.Response, error) {
	qs.Logger.Info(fmt.Sprintf("Getting handler based on opcode %v.", req.GetOpcode()))
	// Select handler for the appropriate operation based on the OpCode
	handler, ok := qs.Handlers[int(req.GetOpcode())]
	if !ok {
		qs.Logger.Info(fmt.Sprintf("Didn't get handler based on opcode %v, defaulting to no-op.", req.GetOpcode()))
		handler = qs.Handlers[0]
	}

	qs.Logger.Info("Starting handler...")

	tStart := time.Now()
	response, err := handler.Handle(req)
	tEnd := time.Now()
	tElapsed := tEnd.Sub(tStart)
	qs.Logger.Info(fmt.Sprintf("Elapsed : [%v], OpCode : [%v]", tElapsed, req.GetOpcode()))
	if err != nil {
		return response, err
	} else {
		return response, nil
	}
}

func (qs *StorageService) HandleErrorResponse(err error) *tcp.Response {
	qs.Logger.Error(err.Error())
	var response = &tcp.Response{}
	// This is the basic failure response status code
	return response
}
