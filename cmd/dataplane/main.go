package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/kokaq/core/internals/logger"
	"github.com/kokaq/server/internals/data"
)

func main() {

	// Define flags
	addr := flag.String("port", "", "Primary server port")
	prootDir := flag.String("rootDir", "", "Root Directory")
	shadress := flag.String("shardManagerAddress", "", "Secondary server port")
	flag.Parse()

	// Fallback to env vars if flags are not set
	shardAddress := *addr
	if shardAddress == "" {
		shardAddress = os.Getenv("PORT")
	}
	if shardAddress == "" {
		shardAddress = "9001" // default fallback
	}

	shardManagerAddress := *shadress
	if shardManagerAddress == "" {
		shardManagerAddress = os.Getenv("SHARD_MANAGER_ADDRESS")
	}
	if shardManagerAddress == "" {
		shardManagerAddress = "8999" // default fallback
	}

	rootDir := *prootDir
	if rootDir == "" {
		rootDir = os.Getenv("ROOT_DIRECTORY")
	}
	if rootDir == "" {
		rootDir = "C://code/kokaq/bin" // default fallback
	}

	logger.ConsoleLog("INFO", "Starting with Shard Port=%s as a child resource of PORT2=%s", shardAddress, shardManagerAddress)

	logger.ConsoleLog("INFO", "Kokaq Data Shard - gRPC Node")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "   Listening on   : 0.0.0.0:%s", shardAddress)
	logger.ConsoleLog("INFO", "   Protocol       : gRPC v1.74.2")
	logger.ConsoleLog("INFO", "   Message Store  : Disk-backed Heap (Primary | Invisibility | DLQ)")
	logger.ConsoleLog("INFO", "   Queue Capacity : Dynamic")
	logger.ConsoleLog("INFO", "   Node Role      : Data Plane (Shard)")
	logger.ConsoleLog("INFO", "")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  System Info")
	logger.ConsoleLog("INFO", "────────────────────────────────────────────")

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

	logger.ConsoleLog("INFO", "────────────────────────────────────────────")
	logger.ConsoleLog("INFO", "  Network Stats")
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

	data.RegisterNode(shardManagerAddress, ":"+shardAddress, "data-plane:"+shardAddress)
	data.StartNode(rootDir, ":"+shardAddress)
	data.UnregisterNode(shardManagerAddress, ":"+shardAddress, "data-plane:"+shardAddress)
}
