package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"./src/gossip"
	"./src/shared"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Usage:", args[0], "port")
		os.Exit(1)
	}
	gossipPort := args[1]
	rand.Seed(time.Now().UnixNano())

	node := shared.NewNode(gossipPort)

	// Connect to Introduction Service
	introdutionConn := gossip.ConnectToIntroduction(node, gossipPort)

	// Start gossip protocol server
	go gossip.StartGossipServer(node, gossipPort)

	// Start SWIM-style failure detection
	go gossip.Ping(node)

	// Receive message from Introduction Service
	gossip.HandleServiceTCPConnection(node, introdutionConn)
}
