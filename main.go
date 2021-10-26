package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/pion/webrtc/v3"
)

var api *webrtc.API //nolint

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.ServeFile(w, r, "./index.html")
		return
	}

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	// Read the offer from the HTTP Post and decode
	offer := webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		panic(err)
	} else if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Set a handler for when a new remote DataChannel is open and when it receives a message
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(m webrtc.DataChannelMessage) {
			fmt.Printf("DataChannel Message:%s\n", m.Data)
		})
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())
		if s == webrtc.PeerConnectionStateFailed {
			if cErr := peerConnection.Close(); cErr != nil {
				fmt.Printf("cannot close peerConnection: %v\n", cErr)
			}

		}
	})

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	<-gatherComplete

	// Encode and respond to the HTTP post with our answer
	if err = json.NewEncoder(w).Encode(peerConnection.LocalDescription()); err != nil {
		panic(err)
	}
}

// sharedUDPConn wraps a net.UDPConn and allows us to intercept ReadFrom calls
// when we get a packet that doesn't appear to be WebRTC we drop that value
type sharedUDPConn struct {
	*net.UDPConn
}

func (s *sharedUDPConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if n, addr, err = s.UDPConn.ReadFrom(p); err == nil {
		if !isWebRTC(p, n) {
			fmt.Printf("Dropped packet that doesn't appear to be WebRTC: %s", p[:n])
			return s.UDPConn.ReadFrom(p)
		}
	}

	return
}

func main() {
	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{0, 0, 0, 0}, Port: 8000})
	if err != nil {
		panic(err)
	}

	// Create a SettingEngine, this allows non-standard WebRTC behavior
	settingEngine := webrtc.SettingEngine{}

	// Configure our SettingEngine to use our UDPMux. By default a PeerConnection has
	// no global state. The API+SettingEngine allows the user to share state between them.
	// In this case we are sharing our listening port across many.
	settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(nil, &sharedUDPConn{udpListener}))

	// Create a new API using our SettingEngine
	api = webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	http.HandleFunc("/", httpHandler)

	fmt.Println("Open http://localhost:8080 to access")
	panic(http.ListenAndServe(":8080", nil))
}

// Matching rules come from
// https://tools.ietf.org/html/rfc7983
func isWebRTC(p []byte, n int) bool {
	if len(p) == 0 {
		return true
	} else if p[0] <= 3 { // STUN
		return true
	} else if p[0] >= 20 && p[0] <= 63 { // DTLS
		return true
	} else if p[0] >= 64 && p[0] <= 79 { // TURN
		return true
	} else if p[0] >= 128 && p[0] <= 191 { // RTP and RTCP
		return true
	}

	return false
}
