package control

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

type ControlServer struct {
	server *internals.KokaqServer
}

type ControlServerConfig struct {
	RootDirectory       string
	Address             string
	ShardManagerAddress string
}

func NewControlServer(telemetryLogger internals.TelemetryLogger, requestTimeout time.Duration) (*ControlServer, error) {
	cleanup := func() {
		logger.ConsoleLog("INFO", "control server cleanup called")
	}
	kokaqServer, err := internals.NewKokaqServer(cleanup, telemetryLogger, requestTimeout)
	return &ControlServer{
		server: kokaqServer,
	}, err
}

func (ds *ControlServer) Start(config ControlServerConfig) error {
	register := func(server *grpc.Server) {
		srv, _ := NewControlPlane(config.RootDirectory, config.ShardManagerAddress)
		proto.RegisterKokaqControlPlaneServer(server, srv)
	}
	err := ds.server.Start(config.Address, register)
	return err
}

func (ds *ControlServer) Stop(ctx context.Context) error {
	err := ds.server.Stop(ctx)
	return err
}

func StartControlServer(address string, shardManagerAddress string) {
	telemetryLogger := &DummyTelemetryLogger{}
	requestTimeout := 0 * time.Second

	cs, err := NewControlServer(telemetryLogger, requestTimeout)
	if err == nil {
		cs.Start(ControlServerConfig{
			RootDirectory:       "",
			Address:             address,
			ShardManagerAddress: shardManagerAddress,
		})
	}
	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cs.Stop(ctx); err != nil {
		logger.ConsoleLog("ERROR", "Error stopping control server: %v", err)
	}
}

type DummyTelemetryLogger struct{}

func (d *DummyTelemetryLogger) LogEvent(event string, data map[string]interface{}) {
	logger.ConsoleLog("INFO", "Telemetry event: %s, data: %v", event, data)
}
