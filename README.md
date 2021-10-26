# pion-webrtc-shared-socket

This example demonstrates how Pion WebRTC can use an already listening UDP socket. On startup we listen on UDP Socket 8000.
We wrap this socket in a `sharedUDPConn`, this `sharedUDPConn` drops `ReadFrom` that don't appear to be WebRTC traffic.

### Running

* `go run main.go`
* Open `http://localhost:8080`

In the command line you should see

```
2021/10/26 14:42:29 Open http://localhost:8080 to access
Peer Connection State has changed: connected
```

This means that the PeerConnection has started and connected succesfully. Now attempt to send non-WebRTC traffic to the process.

* `echo 'Testing' | nc -q 1 -u localhost 800`

This will be printed in the terminal like so

```
Dropped packet that doesn't appear to be WebRTC: Testing
```
