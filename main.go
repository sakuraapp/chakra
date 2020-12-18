package main

import (
	"chakra/pkg/config"
	"chakra/services"
)

func main() {
	envConfig := config.New()
	services.Init(envConfig)
}
