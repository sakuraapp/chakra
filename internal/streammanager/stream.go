package streammanager

import (
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"net"
)

type Stream struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	Port              int  `json:"port"`
	tracks            []*webrtc.TrackLocalStaticRTP
	inbound           []byte
	listener          *net.UDPConn
	allowedHosts      []net.IP
	stopped           chan bool
}

func NewStream(id string, name string, port int) *Stream {
	return &Stream{
		Id:         id,
		Name:       name,
		Port:       port,
		tracks:     []*webrtc.TrackLocalStaticRTP{},
		stopped:    make(chan bool),
	}
}

func (s *Stream) Start() error {
	addr := net.UDPAddr{
		IP: net.ParseIP("0.0.0.0"),
		Port: int(s.Port),
	}

	listener, err := net.ListenUDP("udp", &addr)

	if err != nil {
		return err
	}

	s.listener = listener

	log.Printf("Stream listening on port %v", s.Port)

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "pion")

	if err != nil {
		return err
	}

	s.tracks = append(s.tracks, videoTrack)
	s.inbound = make([]byte, 4096)

	go func() {
		err := s.Handle()

		if err != nil {
			log.WithError(err).Error("Failed to handle incoming data")
		}
	}()

	return nil
}

func (s *Stream) Stop() error {
	s.stopped <- true

	err := s.listener.Close()

	if err != nil {
		return err
	} else {
		log.Printf("Stopped stream on port %v", s.Port)
	}

	return nil
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
