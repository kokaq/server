package shard

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

type ShardServer struct {
	server *internals.KokaqServer
}

type ShardServerConfig struct {
	RootDirectory string
	Address       string
}

func NewShardServer(telemetryLogger internals.TelemetryLogger, requestTimeout time.Duration) (*ShardServer, error) {
	cleanup := func() {
		logger.ConsoleLog("INFO", "shard server cleanup called")
	}
	kokaqServer, err := internals.NewKokaqServer(cleanup, telemetryLogger, requestTimeout)
	return &ShardServer{
		server: kokaqServer,
	}, err
}

func (ds *ShardServer) Start(config ShardServerConfig) error {
	register := func(server *grpc.Server) {
		srv, _ := NewShardPlane(config.RootDirectory)
		proto.RegisterKokaqShardManagerServer(server, srv)
	}
	err := ds.server.Start(config.Address, register)
	return err
}

func (ds *ShardServer) Stop(ctx context.Context) error {
	err := ds.server.Stop(ctx)
	return err
}

func StartShardManager(address string) {
	telemetryLogger := &DummyTelemetryLogger{}
	requestTimeout := 15 * time.Second

	cs, err := NewShardServer(telemetryLogger, requestTimeout)
	if err == nil {
		cs.Start(ShardServerConfig{
			RootDirectory: "",
			Address:       address,
		})
	}
	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cs.Stop(ctx); err != nil {
		logger.ConsoleLog("ERROR", "Error stopping ahard server: %v", err)
	}
}

type DummyTelemetryLogger struct{}

func (d *DummyTelemetryLogger) LogEvent(event string, data map[string]interface{}) {
	logger.ConsoleLog("INFO", "Telemetry event: %s, data: %v", event, data)
}
