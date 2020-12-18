package signalserver

import "github.com/pion/webrtc/v3"

const (
	ACTION_CONNECT = "Connect"
)

type Message struct {
	Action string `json:"action"`
	Stream string `json:"stream"`
	Data   webrtc.SessionDescription `json:"data"`
	Token  string `json:"token"`
}

type OutgoingMessage struct {
	Action string `json:"action"`
	Status int `json:"status"`
	Stream string `json:"stream"`
	Data webrtc.SessionDescription `json:"data"`
}