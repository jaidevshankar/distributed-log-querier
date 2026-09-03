package main

import (
	"testing"
)

func Test(t *testing.T) {
	peers := getPeers()
	peerNumbers := getPeerNumbers()
	port := getPort()
	node := Node{Peers: peers, PeerNumbers: peerNumbers, Me: 0, Port: port}
	node.generateLogFiles()
}