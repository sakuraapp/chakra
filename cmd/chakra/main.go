package main

import (
	"chakra/internal/config"
	"chakra/internal/service"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	strPort := os.Getenv("PORT")
	port, err := strconv.ParseInt(strPort, 10, 64)

	if err != nil {
		log.WithError(err).Fatal("Invalid port")
	}

	nodeId := os.Getenv("NODE_ID")

	if nodeId == "" {
		log.Fatal("NodeId is missing")
	}

	startPort, err := strconv.ParseInt(os.Getenv("START_PORT"), 10, 64)

	if err != nil {
		log.WithError(err).Fatal("StartPort is missing")
	}

	endPort, err := strconv.ParseInt(os.Getenv("END_PORT"), 10, 16)

	if err != nil {
		log.WithError(err).Fatal("EndPort is missing")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDatabase := os.Getenv("REDIS_DATABASE")
	redisDb, _ := strconv.Atoi(redisDatabase)

	conf := &config.Config{
		Port:          int(port),
		NodeId:        nodeId,
		StartPort:     int(startPort),
		EndPort:       int(endPort),
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDatabase: redisDb,
		TLSCertPath:   os.Getenv("TLS_CERT_PATH"),
		TLSKeyPath:    os.Getenv("TLS_KEY_PATH"),
	}

	_, err = service.New(conf)

	if err != nil {
		log.WithError(err).Fatal("Failed to start service")
	}
}