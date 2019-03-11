package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const serverAddr = "172.22.158.32"
const serverPort = "8888"

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println("Usage:", args[0], "name port")
		os.Exit(1)
	}
	name := args[1]
	gossipPort := args[2]

	// Get local IP address
	localIP := getLocalIP()
	if localIP == "" {
		fmt.Println("Error: cannot find local IP.")
		return
	}

	// Connect to Introduction Service
	conn, err := net.Dial("tcp", serverAddr+":"+serverPort)
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "CONNECT "+name+" "+localIP+" "+gossipPort+"\n")
	}

	// Start gossip protocol server
	go startGossipServer(gossipPort)

	// Receive message from Introduction Service
	handleServiceTCPConnection(conn)
}

// Reference https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
// getLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func startGossipServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go handleTCPConnection(conn)
		defer conn.Close()
	}
}

func handleServiceTCPConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		rawMsg, err := reader.ReadString('\n')
		if err == io.EOF {
			fmt.Println("Server offline")
			break
		}

		fmt.Printf(rawMsg)

		// TODO: Add a parse message function
		if strings.HasPrefix(rawMsg, "INTRODUCE") {
			// Handle INTRODUCE
		} else if strings.HasPrefix(rawMsg, "TRANSACTION") {
			// Handle TRANSACTION
		} else if strings.HasPrefix(rawMsg, "DIE") || strings.HasPrefix(rawMsg, "QUIT") {
			break
		} else {
			fmt.Println("Unknown message format.")
		}

	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		rawMsg, err := reader.ReadString('\n')
		if err == io.EOF {
			fmt.Println("Server offline")
			break
		}

		fmt.Printf(rawMsg)

	}
}