package main

import (
	"github.com/kokaq/server/storage"
	"github.com/sirupsen/logrus"
)

const (
	LISTEN_PORT = 9944
)

func main() {
	_, wiresrv, err := storage.NewStorageService(LISTEN_PORT, logrus.New())
	if err != nil {
		panic(err)
	}

	err = wiresrv.Start()
	if err != nil {
		panic(err)
	}
}
