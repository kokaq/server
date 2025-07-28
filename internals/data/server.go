package data

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/protocol/proto"
	"github.com/kokaq/server/internals"
	"google.golang.org/grpc"
)

type DataServer struct {
	server *internals.KokaqServer
}

type DataServerConfig struct {
	RootDirectory string
	Address       string
}

func NewDataServer(telemetryLogger internals.TelemetryLogger, requestTimeout time.Duration) (*DataServer, error) {
	cleanup := func() {
		logger.ConsoleLog("INFO", "data server cleanup called")
	}
	kokaqServer, err := internals.NewKokaqServer(cleanup, telemetryLogger, requestTimeout)
	return &DataServer{
		server: kokaqServer,
	}, err
}

func (ds *DataServer) Start(config DataServerConfig) error {
	register := func(server *grpc.Server) {
		srv, _ := NewDataPlane(config.RootDirectory)
		proto.RegisterKokaqDataPlaneServer(server, srv)
	}
	ds.server.Start(config.Address, register)
	return nil
}

func (ds *DataServer) Stop(ctx context.Context) error {
	ds.server.Stop(ctx)
	return nil
}

func StartDataShard(address string) {
	telemetryLogger := &DummyTelemetryLogger{}
	requestTimeout := 15 * time.Second

	ds, err := NewDataServer(telemetryLogger, requestTimeout)
	if err == nil {
		ds.Start(DataServerConfig{RootDirectory: "C:\\code\\kokaq\\bin", Address: address})
	}
	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ds.Stop(ctx); err != nil {
		logger.ConsoleLog("ERROR", "Error stopping server: %v", err)
	}
}

func RegisterShard(shardManagerAddress string, shardAddress string) {
	conn, err := grpc.NewClient(shardManagerAddress, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		logger.ConsoleLog("ERROR", "Failed to connect to control server: %v", err)
		return
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &proto.RegisterNodeRequest{
		GrpcAddress: shardAddress,
		Shard:       make([]*proto.ShardItem, 0),
	}
	var res *proto.RegisterNodeResponse
	res, err = shardManagerClient.RegisterNode(ctx, req)
	if err != nil {
		logger.ConsoleLog("ERROR", "RegisterNode RPC failed: %v", err)
	}
	if res != nil && res.Accepted {
		logger.ConsoleLog("INFO", "Node Registered")
	}
}

func UnregisterShard(control string, address string) {
	conn, err := grpc.NewClient(control, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		logger.ConsoleLog("INFO", "Failed to connect to control server: %v", err)
		return
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &proto.RegisterNodeRequest{
		GrpcAddress: address,
		Shard:       make([]*proto.ShardItem, 0),
	}
	var res *proto.RegisterNodeResponse
	res, err = shardManagerClient.UnregisterNode(ctx, req)
	if err != nil {
		logger.ConsoleLog("INFO", "UnregisterNode RPC failed: %v", err)
	}
	if res != nil && res.Accepted {
		logger.ConsoleLog("INFO", "Node Unregistered")
	}
}

type DummyTelemetryLogger struct{}

func (d *DummyTelemetryLogger) LogEvent(event string, data map[string]interface{}) {
	logger.ConsoleLog("INFO", "Telemetry event: %s, data: %v", event, data)
}
