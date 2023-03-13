package streammanager

import (
	"chakra/internal/config"
	"errors"
	"github.com/teris-io/shortid"
	"time"
)

var portError = errors.New("no free ports available")

type StreamManager struct {
	conf       *config.Config
	lastPort   int
	freePorts  map[int]bool
	streams    map[string]*Stream
	sid        *shortid.Shortid
	ticker     *time.Ticker
}

func New(conf *config.Config) (*StreamManager, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)

	if err != nil {
		return nil, err
	}

	m := &StreamManager{
		conf:       conf,
		lastPort:   conf.StartPort - 1,
		freePorts:  map[int]bool{},
		sid:        sid,
	}

	return m, nil
}

func (m *StreamManager) CreateStream(name string) (*Stream, error) {
	id, err := m.sid.Generate()

	if err != nil {
		return nil, err
	}

	port, err := m.GetNextPort()

	if err != nil {
		return nil, err
	}

	stream := NewStream(id, name, port)
	m.streams[id] = stream

	return stream, nil
}

func (m *StreamManager) GetNextPort() (int, error) {
	var port int

	if m.lastPort == m.conf.EndPort {
		if len(m.freePorts) > 0 {
			for locPort := range m.freePorts {
				port = locPort
				break
			}

			delete(m.freePorts, port)
		} else {
			return 0, portError
		}
	} else {
		port = m.lastPort + 1
		m.lastPort = port
	}

	return port, nil
}

func (m *StreamManager) GetStream(id string) *Stream {
	return m.streams[id]
}

func (m *StreamManager) GetStreamByName(name string) *Stream {
	for _, stream := range m.streams {
		if stream.Name == name {
			return stream
		}
	}

	return nil
}

func (m *StreamManager) RemoveStream(id string) {
	port := m.streams[id].Port

	delete(m.streams, id)
	m.freePorts[port] = true
}
