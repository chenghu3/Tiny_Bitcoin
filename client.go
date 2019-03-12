package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"./src/shared"
)

const pingTimeout time.Duration = 1500 * time.Millisecond

// Node defination
type Node struct {
	Port         string
	Members      []string
	Transactions []string
	MembersSet   *shared.StringSet
	RWlock       sync.RWMutex
}

// NewNode : construntor for Node struct
func NewNode(port string) *Node {
	node := new(Node)
	node.Port = port
	node.Members = make([]string, 0)
	node.Transactions = make([]string, 0)
	node.MembersSet = shared.NewSet()
	return node
}

// handleServiceTCPConnection: handle messge revieved from introduction service
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
			go joinP2P(node, addr, port)
		} else if strings.HasPrefix(rawMsg, "TRANSACTION") {
			// Handle TRANSACTION
			node.Transactions = append(node.Transactions, rawMsg)
		} else if strings.HasPrefix(rawMsg, "DIE") || strings.HasPrefix(rawMsg, "QUIT") {
			os.Exit(1)
		} else {
			fmt.Println("Unknown message format.")
		}

	}
}

// startGossipServer: set tcp gossip server
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

// handleGossipTCPConnection: handle messge revieved from other peers
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

// sendGossipingMessge: header: "JOIN, TRANSACTION, DEAD", round, message.
// !! add newline in the mesg passed in
func sendGossipingMessge(node *Node, header string, round int, mesg string) {
	gossipMesg := ""
	for {
		node.RWlock.RLock()
		NumMembers := len(node.Members)
		node.RWlock.RUnlock()
		maxRound := int(2 * math.Log(float64(NumMembers)))
		if round > maxRound {
			break
		}
		gossipMesg = header + "," + strconv.Itoa(round) + "," + mesg
		for i := 0; i < 2; i++ {
			// seed in main
			targetPeer := strings.Split(node.Members[rand.Intn(NumMembers)], " ")
			ip := targetPeer[0]
			port := targetPeer[1]
			conn, err := net.Dial("tcp", ip+":"+port)
			// send gossipMesg to peer
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Fprintf(conn, gossipMesg)
			}
			conn.Close()
		}
		round++
	}
}

// Notify existing nodes known from INTRODUCE message this node has join the P2P network.
func joinP2P(node *Node, addr string, port string) {
	conn, err := net.Dial("tcp", addr+":"+port)
	defer conn.Close()
	// send hello to peer
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "HELLO "+node.Port+"\n")
	}
	// receive membershiplist back
	reader := bufio.NewReader(conn)
	membershiplist, err := reader.ReadString('\n')
	if err == io.EOF {
		hostname, _ := net.LookupAddr(addr)
		remotehost := hostname[0]
		machineID := shared.GetNumberFromServerAddress(remotehost)
		fmt.Println("Peer from VM" + strconv.Itoa(machineID) + " in port" + port + " went offline.")
		return
	}
	updateMembershiplist(node, membershiplist)
}

func updateMembershiplist(node *Node, membershiplist string) {
	node.RWlock.Lock()
	defer node.RWlock.Unlock()
	m := strings.Split(membershiplist, "\n")[0]
	members := strings.Split(m, ",")
	for _, member := range members {
		if !node.MembersSet.SetHas(member) {
			node.MembersSet.SetAdd(member)
		}
	}
	node.Members = node.MembersSet.SetToArray()
}

// TODO: change introduction server to known server
func connectToIntroduction(node *Node, gossipPort string) (conn net.Conn) {
	// Get local IP address
	localIP := shared.GetLocalIP()
	if localIP == "" {
		fmt.Println("Error: cannot find local IP.")
		return
	}
	node.Members = append(node.Members, localIP+" "+gossipPort)
	// Connect to Introduction Service
	serverAddr, _ := os.Hostname()
	conn, err := net.Dial("tcp", serverAddr+":8888")
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "CONNECT "+localIP+" "+gossipPort+"\n")
	}
	return
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Usage:", args[0], "port")
		os.Exit(1)
	}
	gossipPort := args[1]
	rand.Seed(time.Now().UnixNano())

	node := NewNode(gossipPort)

	// Connect to Introduction Service
	introdutionConn := connectToIntroduction(node, gossipPort)

	// Start gossip protocol server
	go startGossipServer(node, gossipPort)

	// Receive message from Introduction Service
	handleServiceTCPConnection(node, introdutionConn)
}
