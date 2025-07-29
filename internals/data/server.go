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
	"google.golang.org/grpc/credentials/insecure"
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

func StartNode(rootDirectory string, address string) {
	telemetryLogger := &DummyTelemetryLogger{}
	requestTimeout := 15 * time.Second

	ds, err := NewDataServer(telemetryLogger, requestTimeout)
	if err == nil {
		ds.Start(DataServerConfig{RootDirectory: rootDirectory, Address: address})
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

func RegisterNode(shardManagerAddress string, shardAddress string, shardInternalAddress string) {
	maxRetries := 10
	retryDelay := 2 * time.Second

	var conn *grpc.ClientConn
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.ConsoleLog("INFO", "Attempt %d: Connecting to shard manager at %s", attempt, shardManagerAddress)
		logger.ConsoleLog("INFO", "Dialing gRPC target: %s", shardManagerAddress)
		conn, err = grpc.NewClient(shardManagerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			logger.ConsoleLog("INFO", "Connected to shard manager")
			break
		}

		logger.ConsoleLog("WARN", "Connection failed: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}

	if conn == nil {
		logger.ConsoleLog("ERROR", "Failed to connect to shard manager after %d attempts", maxRetries)
		return
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &proto.RegisterNodeRequest{
		GrpcAddress:     shardAddress,
		InternalAddress: shardInternalAddress,
		Shard:           make([]*proto.ShardItem, 0),
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

func UnregisterNode(shardManagerAddress string, shardAddress string, shardInternalAddress string) {
	conn, err := grpc.NewClient(shardManagerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		logger.ConsoleLog("INFO", "Failed to connect to shard manager: %v", err)
		return
	}
	defer conn.Close()

	shardManagerClient := proto.NewKokaqShardManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &proto.RegisterNodeRequest{
		GrpcAddress:     shardAddress,
		InternalAddress: shardInternalAddress,
		Shard:           make([]*proto.ShardItem, 0),
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
