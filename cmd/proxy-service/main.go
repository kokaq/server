package main

import (
	"context"

	"github.com/kokaq/server/internals/core/http"
	"github.com/kokaq/server/proxy"
	"github.com/sirupsen/logrus"
)

const (
	LISTEN_PORT = 9943
)

func main() {

	// Create an instance of queue service
	var proxyConfig = &proxy.ProxyServiceConfig{
		HttpServerConfig: http.KokaqHttpServerConfig{
			Port: LISTEN_PORT,
		},
		Logger: logrus.New(),
	}
	proxyService := proxy.NewProxyService(*proxyConfig)
	err := proxyService.Start(context.Background())
	if err != nil {
		panic(err)
	}
}
