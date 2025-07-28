package main

import (
	"flag"

	"github.com/kokaq/server/internals/data"
)

func main() {

	// Define a flag for port
	addr := flag.String("port", "9002", "TCP port to listen on")
	shadress := flag.String("shard", "9000", "")
	flag.Parse()

	data.RegisterShard(":"+*shadress, ":"+*addr)
	data.StartDataShard(":" + *addr)
	data.UnregisterShard(":"+*shadress, ":"+*addr)
}
