package config

type Config struct {
	Port int
	NodeId string
	StartPort int
	EndPort int
	RedisAddr string
	RedisPassword string
	RedisDatabase int
	TLSCertPath string
	TLSKeyPath string
}