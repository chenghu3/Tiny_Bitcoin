package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"./src/shared"
)

const gossipInterval time.Duration = 500 * time.Millisecond
const pingInterval time.Duration = 300 * time.Millisecond

// Node defination
type Node struct {
	Port              string
	Transactions      *shared.StringSet
	TransactionBuffer *shared.MsgBuffer
	MembersSet        *shared.StringSet
	FailMsgBuffer     *shared.MsgBuffer
}

// NewNode : construntor for Node struct
func NewNode(port string) *Node {
	node := new(Node)
	node.Port = port
	node.Transactions = shared.NewSet()
	node.TransactionBuffer = shared.NewMsgBuffer(25)
	node.MembersSet = shared.NewSet()
	node.FailMsgBuffer = shared.NewMsgBuffer(10)
	return node
}

// handleServiceTCPConnection: handle messge revieved from introduction service
func handleServiceTCPConnection(node *Node, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		rawMsg, err := reader.ReadString('\n')
		rawMsg = strings.Trim(rawMsg, "\n")

		if err == io.EOF {
			fmt.Println("Server offline")
			break
		}

		fmt.Println("SERVICE:" + rawMsg)

		// TODO: Add a parse message function
		if strings.HasPrefix(rawMsg, "INTRODUCE") {
			// self introduction
			arr := strings.Split(rawMsg, " ")
			addr := arr[1]
			port := arr[2]
			go joinP2P(node, addr, port)
		} else if strings.HasPrefix(rawMsg, "TRANSACTION") {
			// Handle TRANSACTION
			node.Transactions.SetAdd(rawMsg)
			node.TransactionBuffer.Add(rawMsg)
			logWithTimestamp(rawMsg)
			// go sendGossipingMsg(node, "TRANSACTION", 0, rawMsg)
		} else if strings.HasPrefix(rawMsg, "DIE") || strings.HasPrefix(rawMsg, "QUIT") {
			os.Exit(1)
		} else {
			fmt.Println("Unknown message format.")
		}

	}
}

func logWithTimestamp(rawMsg string) {
	fmt.Println("LOG " + time.Now().Format("2006-01-02 15:04:05.000000") + " " + rawMsg)
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
		// fmt.Println(conn.RemoteAddr().String())
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

	stop := false
	for !stop {
		gossipRawMsg, err := reader.ReadString('\n')
		stop = err == io.EOF
		gossipRawMsg = strings.Trim(gossipRawMsg, "\n")
		// fmt.Println("RECEIVING", gossipRawMsg)

		if strings.HasPrefix(gossipRawMsg, "HELLO") {
			addrPort := addr + " " + strings.Split(gossipRawMsg, " ")[1]
			node.MembersSet.SetAdd(addrPort)
			memberString := strings.Join(node.MembersSet.SetToArray(), ",") + "\n"
			fmt.Fprintf(conn, memberString)
			// send JOIN msg
			go sendGossipingMsg(node, "JOIN", 0, addrPort)
		} else if strings.HasPrefix(gossipRawMsg, "JOIN") {
			round, rawMsg := ParseGossipingMessage(gossipRawMsg)
			if node.MembersSet.SetAdd(rawMsg) {
				go sendGossipingMsg(node, "JOIN", round+1, rawMsg)
				fmt.Println("New User:" + rawMsg)
				fmt.Println("MEMBERLIST SIZE" + strconv.Itoa(node.MembersSet.Size()))
			}
		} else if strings.HasPrefix(gossipRawMsg, "TRANSACTION") {
			// _, rawMsg := ParseGossipingMessage(gossipRawMsg)
			rawMsg := gossipRawMsg
			if node.Transactions.SetAdd(rawMsg) {
				// First time infected
				logWithTimestamp(rawMsg)
				// go sendGossipingMsg(node, "TRANSACTION", round+1, rawMsg)
				node.TransactionBuffer.Add(rawMsg)
			}
		} else if strings.HasPrefix(gossipRawMsg, "DEAD") {
			if len(gossipRawMsg) > 5 {
				failList := strings.Split(gossipRawMsg[5:], ",")
				for _, machine := range failList {
					fmt.Println("SWIM RECEIVED " + machine)
					if node.MembersSet.SetDelete(machine) {
						fmt.Println("SWIM DELETE " + machine)
						node.FailMsgBuffer.Add(machine)
					}
				}
			}
		} else {
			fmt.Print("Unknown gossip message format:")
			fmt.Println(gossipRawMsg)
		}
	}
}

// ParseGossipingMessage : Parse a gossip message, return the current round and the message body
func ParseGossipingMessage(gossipRawMsg string) (int, string) {
	params := strings.Split(gossipRawMsg, ",")
	round, _ := strconv.Atoi(params[1])
	rawMsg := params[2]
	return round, rawMsg
}

// sendGossipingMsg: header: "JOIN, TRANSACTION, DEAD", round, message.
// !! add newline in the mesg passed in
func sendGossipingMsg(node *Node, header string, round int, mesg string) {
	gossipMesg := ""
	for {
		NumMembers := node.MembersSet.Size()
		maxRound := int(4 * math.Log(float64(NumMembers))) // TODO: Change to a constant
		if round > maxRound {
			break
		}
		gossipMesg = header + "," + strconv.Itoa(round) + "," + mesg + "\n"
		for i := 0; i < 2; i++ {
			// seed in main
			target := node.MembersSet.GetRandom()
			targetPeer := strings.Split(target, " ")
			ip := targetPeer[0]
			port := targetPeer[1]
			conn, err := net.Dial("tcp", ip+":"+port)
			// send gossipMesg to peer
			if err != nil {
				// failure detected!
				i--
			} else {
				fmt.Fprintf(conn, gossipMesg)
				conn.Close()
			}
		}
		round++
		time.Sleep(gossipInterval)
	}
}

func handleDialFail(node *Node, peer string) {
	node.MembersSet.SetDelete(peer)
	go sendGossipingMsg(node, "DEAD", 0, peer)
}

// ping : SWIM style dissemination of membership updates
func ping(node *Node) {
	for {
		time.Sleep(pingInterval)
		target := node.MembersSet.GetRandom()
		if target == "" {
			continue
		}
		targetPeer := strings.Split(target, " ")
		ip := targetPeer[0]
		port := targetPeer[1]
		conn, err := net.Dial("tcp", ip+":"+port)
		if err != nil {
			// failure detected!
			if strings.HasSuffix(err.Error(), "connect: connection refused") {
				node.MembersSet.SetDelete(target)
				node.FailMsgBuffer.Add(target)
				fmt.Println("FAILURE DETECTED " + target)
			} else {
				fmt.Println("Dial Error: ", err)
			}
		} else {
			// SWIM Implementation would send membership update message here
			swimMsg := "DEAD " + strings.Join(node.FailMsgBuffer.GetN(10), ",") + "\n"
			fmt.Fprintf(conn, swimMsg)
			fmt.Print("SWIM SENT " + swimMsg)
			transactionsMsg := strings.Join(node.TransactionBuffer.GetN(10000), "\n") + "\n"
			fmt.Fprintf(conn, transactionsMsg)
			// fmt.Print("TRANSACTIONS " + transactionsMsg)
			conn.Close()
		}
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
	m := strings.Trim(membershiplist, "\n")
	members := strings.Split(m, ",")
	for _, member := range members {
		if !node.MembersSet.SetHas(member) {
			node.MembersSet.SetAdd(member)
		}
	}
}

// TODO: change introduction server to known server
func connectToIntroduction(node *Node, gossipPort string) (conn net.Conn) {
	// Get local IP address
	localIP := shared.GetLocalIP()
	if localIP == "" {
		fmt.Println("Error: cannot find local IP.")
		return
	}
	// node.Members = append(node.Members, localIP+" "+gossipPort)
	node.MembersSet.SetAdd(localIP + " " + gossipPort)
	// Connect to Introduction Service
	serverAddr := "172.22.158.32"
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

	// Start SWIM-style failure detection
	go ping(node)

	// Receive message from Introduction Service
	handleServiceTCPConnection(node, introdutionConn)
}
