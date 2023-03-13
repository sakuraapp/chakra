package chakra

import "github.com/pion/webrtc/v3"

type MessageType int

const (
	MessageTypeCreate MessageType = iota
	MessageTypeSignal
)

type Message struct {
	Type MessageType `msgpack:"type"`
	Data interface{}
}

type CreatePeerRequest struct {
	StreamId string `msgpack:"stream_id"`
	SessionId string `msgpack:"session_id"`
	Offer webrtc.SessionDescription `msgpack:"offer"`
}