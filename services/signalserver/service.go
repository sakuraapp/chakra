package signalserver

import (
	"chakra/pkg/config"
	"chakra/services/streammanager"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type SignalServer struct {
	config        config.ConfigSource
	streamManager *streammanager.StreamManager
	port          int
}

type PostData struct {
	Token        string
	Name         string
	AllowedHosts []string
}

func New(config config.ConfigSource, streamManager *streammanager.StreamManager) *SignalServer {
	port, err := config.GetInt("SIGNAL_PORT")

	if err != nil {
		panic(err)
	}

	server := SignalServer{
		port:          port,
		config:        config,
		streamManager: streamManager,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", server.ServeHome)
	r.HandleFunc("/streams", server.CreateStream)

	http.Handle("/", r)

	return &server
}

func (server *SignalServer) CreateStream(w http.ResponseWriter, r *http.Request) {
	var body PostData
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	token := ""

	reqToken := r.Header.Get("Authorization")

	if len(reqToken) > 0 {
		tokenParts := strings.Split(reqToken, "Bearer ")

		if len(tokenParts) > 1 {
			token = tokenParts[1]
		}
	}

	authToken := server.config.GetString("AUTH_TOKEN")

	if len(authToken) > 0 && token != authToken {
		w.WriteHeader(401)
		return
	}

	name := body.Name

	if len(name) == 0 {
		w.WriteHeader(400)
		return
	}

	switch r.Method {
	case "POST":
		existingStream := server.streamManager.GetStreamByName(name)

		if existingStream != nil {
			w.WriteHeader(423)
			return
		}

		allowedHosts := make([]net.IP, 0)

		if streammanager.VerifyHosts {
			if len(body.AllowedHosts) > 0 {
				inputHosts := body.AllowedHosts

				for i := 0; i < len(inputHosts); i++ {
					ip := net.ParseIP(inputHosts[i])

					if ip != nil {
						allowedHosts = append(allowedHosts, ip)
					}
				}
			} else {
				host, _, err := net.SplitHostPort(r.RemoteAddr)

				if err != nil {
					w.WriteHeader(500)
					return
				}

				if host != "::1" {
					allowedHosts = append(allowedHosts)
				}
			}
		}

		stream, err := server.streamManager.CreateStream(name, allowedHosts)

		if err != nil {
			panic(err)
		}

		stream.Start()

		if err = json.NewEncoder(w).Encode(stream); err != nil {
			panic(err)
		}
	case "DELETE":
		if server.streamManager.RemoveStream(name) {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
	}
}

func (server *SignalServer) Listen() {
	addr := fmt.Sprintf("0.0.0.0:%v", server.port)
	done := make(chan bool)

	go func() {
		err := http.ListenAndServe(addr, nil)

		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()

	log.Printf("Listening on port %v", server.port)
	<-done
}

func (server *SignalServer) ServeHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	serveWs(server.streamManager, w, r)
}
