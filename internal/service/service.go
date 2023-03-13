package service

import (
	"chakra/internal/config"
	"chakra/internal/streammanager"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	chakrapb "github.com/sakuraapp/protobuf/chakra"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

const chakraKey = "chakra.%v"
const maxConnectionAge = 5 * time.Minute

type Server struct {
	chakrapb.UnimplementedChakraServiceServer
	conf *config.Config
	rdb *redis.Client
	streamMgr *streammanager.StreamManager
}

func (s *Server) CreateStream(ctx context.Context, req *chakrapb.CreateRequest) (*chakrapb.StreamInfo, error) {
	name := req.Name
	stream, err := s.streamMgr.CreateStream(name)

	if err != nil {
		return nil, err
	}

	return &chakrapb.StreamInfo{
		Id: stream.Id,
		NodeId: s.conf.NodeId,
	}, nil
}

func New(conf *config.Config) (*Server, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: conf.RedisAddr,
		Password: conf.RedisPassword,
		DB: conf.RedisDatabase,
	})

	streamMgr, err := streammanager.New(conf)

	if err != nil {
		return nil, err
	}

	s := &Server{
		rdb: rdb,
		streamMgr: streamMgr,
	}

	creds, err := credentials.NewServerTLSFromFile(conf.TLSCertPath, conf.TLSKeyPath)

	if err != nil {
		log.WithError(err).Fatal("Failed to load SSL/TLS key pair")
	}

	addr := fmt.Sprintf("0.0.0.0:%v", conf.Port)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		log.WithError(err).Fatal("Failed to start TCP server")
	}

	log.Printf("Listening on port %v", conf.Port)

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: maxConnectionAge,
		}),
	}

	grpcServer := grpc.NewServer(opts...)
	chakrapb.RegisterChakraServiceServer(grpcServer, s)
	err = grpcServer.Serve(listener)

	if err != nil {
		log.WithError(err).Fatal("Failed to start gRPC server")
	}

	return s, nil
}