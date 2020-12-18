package services

import (
	"chakra/pkg/config"
	"chakra/services/signalserver"
	"chakra/services/streammanager"
)

type Services struct {
	SignalServer  *signalserver.SignalServer
	StreamManager *streammanager.StreamManager
}

func Init(config config.ConfigSource) {
	done := make(chan bool)

	services := Services{}
	services.StreamManager = streammanager.New(config)
	services.SignalServer = signalserver.New(config, services.StreamManager)

	go services.SignalServer.Listen()

	<-done
}
