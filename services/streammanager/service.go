package streammanager

import (
	"chakra/pkg/config"
	"errors"
	"net"
	"time"

	"github.com/teris-io/shortid"
)

const (
	// how often the manager should check for inactive streams
	tickRate = 60 * time.Second

	// how often a stream can stay online without receiving any new frames
	maxIdlePeriod = 5 * time.Minute

	// whether or not to verify if hosts are authorized
	VerifyHosts = true
)

type StreamManager struct {
	startPort int
	endPort   int
	portMap   map[int]bool
	streams   []*Stream
	sid       *shortid.Shortid
	ticker    *time.Ticker
}

func New(config config.ConfigSource) *StreamManager {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)

	if err != nil {
		panic(err)
	}

	startPort, err := config.GetInt("START_PORT")

	if err != nil {
		panic(err)
	}

	endPort, err := config.GetInt("END_PORT")

	if err != nil {
		panic(err)
	}

	manager := &StreamManager{
		startPort: startPort,
		endPort:   endPort,
		portMap:   map[int]bool{},
		streams:   make([]*Stream, 0),
		sid:       sid,
	}

	go manager.SetupTicker()

	return manager
}

// This ticker will eliminate inactive streams
func (mgr *StreamManager) SetupTicker() {
	mgr.ticker = time.NewTicker(tickRate)

	defer func() {
		mgr.ticker.Stop()
	}()

	for {
		select {
		case <-mgr.ticker.C:
			for i := 0; i < len(mgr.streams); i++ {
				stream := mgr.streams[i]

				if stream.ActiveConnections < 1 && time.Now().Sub(stream.lastFrame) > maxIdlePeriod {
					mgr.streams = append(mgr.streams[:i], mgr.streams[i+1:]...)
					stream.Stop()
					i--
				}
			}
		}
	}
}

func (mgr *StreamManager) CreateStream(name string, allowedHosts []net.IP) (*Stream, error) {
	id, err := mgr.sid.Generate()

	if err != nil {
		return nil, err
	}

	port, err := mgr.GetNextPort()
	stream := NewStream(id, name, port, allowedHosts)

	if err != nil {
		return nil, err
	}

	mgr.streams = append(mgr.streams, stream)

	return stream, nil
}

func (mgr *StreamManager) GetNextPort() (int, error) {
	lastPort := mgr.startPort - 1
	mappedPorts := len(mgr.portMap)

	if mappedPorts > 0 {
		for port, status := range mgr.portMap {
			if !status {
				lastPort = port - 1
				break
			} else if port == mappedPorts-1 {
				lastPort = port
			}
		}
	}

	if lastPort < mgr.endPort {
		return lastPort + 1, nil
	} else {
		return 0, errors.New("no ports available")
	}
}

func (mgr *StreamManager) GetStream(id string) *Stream {
	for _, stream := range mgr.streams {
		if stream.Id == id {
			return stream
		}
	}

	return nil
}

func (mgr *StreamManager) GetStreamByName(name string) *Stream {
	for _, stream := range mgr.streams {
		if stream.Name == name {
			return stream
		}
	}

	return nil
}

func (mgr *StreamManager) RemoveStream(name string) bool {
	for i, stream := range mgr.streams {
		if stream.Name == name {
			mgr.streams = append(mgr.streams[:i], mgr.streams[i+1:]...)
			stream.Stop()

			return true
		}
	}

	return false
}
