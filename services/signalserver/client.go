package signalserver

import (
	"bytes"
	"chakra/services/streammanager"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 6072
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	streamManager *streammanager.StreamManager
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)

			if err != nil {
				return
			}

			_, _ = w.Write(message)
			n := len(c.send)

			for i := 0; i < n; i++ {
				_, _ = w.Write(newline)
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(str string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}

			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		go c.ProcessMessage(message)
	}
}

func (c *Client) ProcessMessage(msg []byte) {
	var message Message
	err := json.Unmarshal(msg, &message)

	if err != nil {
		log.Print("Invalid message: ", string(msg[:]))
		return
	}

	if len(message.Action) == 0 || len(message.Stream) == 0 {
		return
	}

	stream := c.streamManager.GetStreamByName(message.Stream)

	if stream == nil {
		_ = c.Send(OutgoingMessage{
			Status: 404,
			Stream: message.Stream,
		})
		return
	}

	//log.Print(	"message: ", string(msg[:]))
	log.Printf("message: %+v", message)

	switch message.Action {
	case ACTION_CONNECT:
		offer := message.Data

		answer, err := stream.CreatePeer(offer)

		if err != nil {
			c.Status(500)
			log.Print(err)
			return
		}

		err = c.Send(OutgoingMessage{
			Action: ACTION_CONNECT,
			Stream: stream.Name,
			Data: *answer,
		})

		if err != nil {
			panic(err)
		}
	}
}

func (c *Client) Send(message OutgoingMessage) error {
	if message.Status == 0 {
		message.Status = 200
	}

	output, err := json.Marshal(message)

	if err != nil {
		return err
	}

	c.send <- output

	return nil
}

func (c *Client) Status(code int) {
	err := c.Send(OutgoingMessage{Status: code})

	if err != nil {
		panic(err)
	}
}

func serveWs(streamManager *streammanager.StreamManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	client := Client{
		conn: conn,
		send: make(chan []byte, 256),
		streamManager: streamManager,
	}

	go client.WritePump()
	go client.ReadPump()
}
