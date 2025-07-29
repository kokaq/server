package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/server/internals/control"
	"github.com/kokaq/server/internals/shard"
)

func main() {
	// ports := flag.String("shard", "8999", "TCP port to listen on")
	// flag.Parse()
	// portc := flag.String("control", "9000", "TCP port to listen on")
	// flag.Parse()

	// Define flags
	paddr := flag.String("port", "", "Primary server port")
	pshadress := flag.String("shardManagerAddress", "", "Secondary server port")
	flag.Parse()

	// Fallback to env vars if flags are not set
	port1 := *paddr
	if port1 == "" {
		port1 = os.Getenv("PORT")
	}
	if port1 == "" {
		port1 = "9000" // default fallback
	}

	port2 := *pshadress
	if port2 == "" {
		port2 = os.Getenv("SHARD_MANAGER_ADDRESS")
	}
	if port2 == "" {
		port2 = "8999" // default fallback
	}

	logger.ConsoleLog("INFO", "Starting with Shard Port=%s as a child resource of PORT2=%s", port1, port2)

	logger.ConsoleLog("INFO", "Kokaq Control Plane")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "   Listening on   : 0.0.0.0:%s", port1)
	logger.ConsoleLog("INFO", "   Protocol       : gRPC v1.74.2")
	logger.ConsoleLog("INFO", "   Message Store  : Disk-backed Heap (Primary | Invisibility | DLQ)")
	logger.ConsoleLog("INFO", "   Queue Capacity : Dynamic")
	logger.ConsoleLog("INFO", "   Node Role      : Control Plane + Shard Manager")
	logger.ConsoleLog("INFO", "")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  System Info")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")

	logger.ConsoleLog("INFO", "  Go Version    : %s", runtime.Version())
	logger.ConsoleLog("INFO", "  Build Commit  : %s | %s", "92b134ac", time.Now().Format(time.RFC3339))

	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  Runtime Stats (Startup)")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	logger.ConsoleLog("INFO", "  Uptime        : %s", "0s")
	logger.ConsoleLog("INFO", "  Heap Alloc    : %.2f MB", float64(memStats.HeapAlloc)/(1024*1024))
	logger.ConsoleLog("INFO", "  GC Cycles    : %d", memStats.NumGC)
	logger.ConsoleLog("INFO", "  Goroutines   : %d", runtime.NumGoroutine())

	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  Disk Stats")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")

	// logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	// logger.ConsoleLog("INFO", "  Shard State")
	// logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	// logger.ConsoleLog("INFO", "  Registered Queues : %d", registeredQueues)
	// logger.ConsoleLog("INFO", "   Total Messages    : %d", totalMessages)

	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  Telemetry      : Enabled")
	// logger.ConsoleLog("INFO", "  Tracing        : Jaeger [http://localhost:16686]")
	// logger.ConsoleLog("INFO", "  Auth Mode      : Token-based")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "")
	logger.ConsoleLog("INFO", "Data Shard is up and humming... awaiting RPCs.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		shard.StartShardManager(":" + port2)
	}()

	go func() {
		control.StartControlServer(":"+port1, ":"+port2)
	}()

	<-ctx.Done()
}
