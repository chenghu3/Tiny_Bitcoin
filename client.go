package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const serverAddr = "172.22.158.32"
const serverPort = "8888"
const pingTimeout time.Duration = 1500 * time.Millisecond

// Node defination
type Node struct {
	Port         string
	Members      []string
	Transactions []string
}

// NewNode : construntor for Node struct
func NewNode(port string) *Node {
	node := new(Node)
	node.Port = port
	node.Members = make([]string, 0)
	node.Transactions = make([]string, 0)
	return node
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

func handleServiceTCPConnection(node *Node, conn net.Conn) {
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
			idx := strings.Index(rawMsg, " ") + 1
			node.Members = append(node.Members, rawMsg[idx:len(rawMsg)-1])
			fmt.Println("Current Members:", node.Members)
			// self introduction
			arr := strings.Split(rawMsg, " ")
			addr := arr[1]
			port := arr[2]
			port = port[:len(port)-1]
			go join(node, addr, port)
		} else if strings.HasPrefix(rawMsg, "TRANSACTION") {
			// Handle TRANSACTION
			node.Transactions = append(node.Transactions, rawMsg)
		} else if strings.HasPrefix(rawMsg, "DIE") || strings.HasPrefix(rawMsg, "QUIT") {
			break
		} else {
			fmt.Println("Unknown message format.")
		}

	}
}

func startGossipServer(node *Node, port string) {
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
		go handleGossipTCPConnection(node, conn)
		defer conn.Close()
	}
}

func handleGossipTCPConnection(node *Node, conn net.Conn) {
	defer conn.Close()

	addr := strings.Split(conn.RemoteAddr().String(), ":")[0]

	reader := bufio.NewReader(conn)

	for {
		rawMsg, err := reader.ReadString('\n')
		if err == io.EOF {
			fmt.Println("Server offline")
			break
		}

		fmt.Printf(rawMsg)
		if strings.HasPrefix(rawMsg, "HELLO") {
			node.Members = append(node.Members, addr+" "+strings.Split(rawMsg, " ")[1])
			memberString := strings.Join(node.Members, ",") + "\n"
			fmt.Fprintf(conn, memberString)
		}
	}
}

func join(node *Node, addr string, port string) {
	conn, err := net.Dial("tcp", addr+":"+port)
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "HELLO "+node.Port+"\n")
	}
	reader := bufio.NewReader(conn)
	rawMsg, err := reader.ReadString('\n')
	if err == io.EOF {
		fmt.Println("Node offline")
		return
	}
	fmt.Printf(rawMsg)
	m := strings.Split(rawMsg, "\n")[0]
	members := strings.Split(m, ",")
	node.Members = members
	fmt.Println(node.Members)
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Usage:", args[0], "port")
		os.Exit(1)
	}
	gossipPort := args[1]

	node := NewNode(gossipPort)

	// Get local IP address
	localIP := getLocalIP()
	if localIP == "" {
		fmt.Println("Error: cannot find local IP.")
		return
	}

	node.Members = append(node.Members, localIP+" "+gossipPort)

	// Connect to Introduction Service
	conn, err := net.Dial("tcp", serverAddr+":"+serverPort)
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "CONNECT "+localIP+" "+gossipPort+"\n")
	}

	// Start gossip protocol server
	go startGossipServer(node, gossipPort)

	// Receive message from Introduction Service
	handleServiceTCPConnection(node, conn)
}
