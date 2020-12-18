package streammanager

import (
	"log"
	"net"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Stream struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	Port              int    `json:"port"`
	tracks            []*webrtc.TrackLocalStaticRTP
	inbound           []byte
	listener          *net.UDPConn
	allowedHosts      []net.IP
	stopped           chan bool
	lastFrame         time.Time
	ActiveConnections int
}

func NewStream(id string, name string, port int, allowedHosts []net.IP) *Stream {
	return &Stream{
		Id:                id,
		Name:              name,
		Port:              port,
		tracks:            make([]*webrtc.TrackLocalStaticRTP, 0),
		allowedHosts:      allowedHosts,
		stopped:           make(chan bool),
		lastFrame:         time.Now(),
		ActiveConnections: 0,
	}
}

func (s *Stream) Start() {
	addr := net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: s.Port}

	listener, err := net.ListenUDP("udp", &addr)

	s.listener = listener

	if err != nil {
		panic(err)
	}

	log.Printf("Stream listening on port %v", s.Port)

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "pion")

	if err != nil {
		panic(err)
	}

	s.tracks = append(s.tracks, videoTrack)
	s.inbound = make([]byte, 4096)

	go s.Read()
	go s.Handle()
}

func (s *Stream) Stop() {
	if err := s.listener.Close(); err != nil {
		panic(err)
	} else {
		log.Print("Stopped stream on port ", s.Port)
	}

	s.stopped <- true
}

func (s *Stream) CreatePeer(offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, track := range s.tracks {
		rtpSender, err := peerConnection.AddTrack(track)

		if err != nil {
			return nil, err
		}

		go s.HandleRTCP(rtpSender)
	}

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		currentState := state.String()

		if currentState == "connected" {
			s.ActiveConnections++
		} else if currentState == "disconnected" {
			s.ActiveConnections--
		}

		log.Print(currentState)
	})

	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	answer, err := peerConnection.CreateAnswer(nil)

	if err != nil {
		return nil, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

func (s *Stream) HandleRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)

	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}

func (s *Stream) IsAllowed(ip net.IP) bool {
	if !VerifyHosts {
		return true
	}

	for _, addr := range s.allowedHosts {
		if ip.Equal(addr) {
			return true
		}
	}

	return false
}

func (s *Stream) Read() {
	n, addr, err := s.listener.ReadFromUDP(s.inbound)

	if addr != nil && s.IsAllowed(addr.IP) {
		s.lastFrame = time.Now()

		if err != nil {
			if <-s.stopped {
				return
			} else {
				panic(err)
			}
		}

		packet := &rtp.Packet{}

		if err = packet.Unmarshal(s.inbound[:n]); err != nil {
			panic(err)
		}
	}
}

func (s *Stream) Handle() error {
	for {
		select {
		case <-s.stopped:
			return nil
		default:
			n, _, err := s.listener.ReadFrom(s.inbound)

			if err != nil {
				return err
			}

			if _, writeErr := s.tracks[0].Write(s.inbound[:n]); writeErr != nil {
				return err
			}
		}
	}
}
