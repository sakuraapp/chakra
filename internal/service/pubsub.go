package service

import (
	"chakra/internal/chakra"
	"context"
	"fmt"
	"github.com/sakuraapp/pubsub"
	"github.com/sakuraapp/shared/pkg/dispatcher"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
)

func (a *Server) initPubsub()  {
	ctx := context.Background()

	chName := fmt.Sprintf(chakraKey, a.conf.NodeId)
	ps := a.rdb.Subscribe(ctx, chName)

	d := pubsub.NewRedisDispatcher(ctx, nil, "chakra", a.rdb)

	go func() {
		for {
			message, err := ps.ReceiveMessage(ctx)

			if err != nil {
				log.WithError(err).Error("PubSub Error")
				continue
			}

			var msg chakra.Message

			err = msgpack.Unmarshal([]byte(message.Payload), &msg)

			if err != nil {
				log.WithError(err).Error("PubSub Deserialization Error")
				continue
			}

			switch msg.Type {
			case chakra.MessageTypeCreate:
				data, ok := msg.Data.(chakra.CreatePeerRequest)

				if !ok {
					continue
				}

				s := a.streamMgr.GetStream(data.StreamId)
				desc, err := s.CreatePeer(data.Offer)

				if err != nil {
					log.WithError(err).Error("Failed to create a new peer")
					continue
				}

				d.Dispatch(&pubsub.Message{
					Target: pubsub.MessageTarget{

					}
				})
			}
		}
	}()
}