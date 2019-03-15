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
	"sync"
	"time"

	"./src/shared"
)

const gossipInterval time.Duration = 500 * time.Millisecond
const failureInterval time.Duration = 1 * time.Second

// Node defination
type Node struct {
	Port         string
	Members      []string
	Transactions *shared.StringSet
	MembersSet   *shared.StringSet
	mutex        sync.Mutex
}

// NewNode : construntor for Node struct
func NewNode(port string) *Node {
	node := new(Node)
	node.Port = port
	node.Members = make([]string, 0)
	node.Transactions = shared.NewSet()
	node.MembersSet = shared.NewSet()
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
			logWithTimestamp(rawMsg)
			go sendGossipingMsg(node, "TRANSACTION", 0, rawMsg)
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

	gossipRawMsg, _ := reader.ReadString('\n')
	gossipRawMsg = strings.Trim(gossipRawMsg, "\n")

	// fmt.Printf(gossipRawMsg)
	if strings.HasPrefix(gossipRawMsg, "HELLO") {
		// node.Members = append(node.Members, addr+" "+strings.Split(gossipRawMsg, " ")[1])
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
			// fmt.Print("Updated Membership List")
			fmt.Println(node.MembersSet.SetToArray())
		}
	} else if strings.HasPrefix(gossipRawMsg, "TRANSACTION") {
		round, rawMsg := ParseGossipingMessage(gossipRawMsg)
		if node.Transactions.SetAdd(rawMsg) {
			// First time infected
			logWithTimestamp(rawMsg)
			go sendGossipingMsg(node, "TRANSACTION", round+1, rawMsg)
		}
	} else if strings.HasPrefix(gossipRawMsg, "DEAD") {
		// if node.MembersSet.SetHas(rawMsg) {
		// 	fmt.Print("DEAD list before: ")
		// 	fmt.Println(node.MembersSet.SetToArray())
		// }
		params := strings.Split(gossipRawMsg, ",")
		rawMsg := params[1]
		// node.mutex.Lock()
		if node.MembersSet.SetHas(rawMsg) {
			node.MembersSet.SetDelete(rawMsg)
		}
		// node.mutex.Unlock()
		// 	fmt.Print("DEAD list after: ")
		// 	fmt.Println(node.MembersSet.SetToArray())
		// 	go sendGossipingMsg(node, "DEAD", round+1, rawMsg)
		// }
	} else if strings.HasPrefix(gossipRawMsg, "PING") {
		return
	} else {
		fmt.Print("Unknown gossip message format:")
		fmt.Println(gossipRawMsg)
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
		maxRound := int(20 * math.Log(float64(NumMembers))) // TODO: Change to a constant
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
				// if strings.HasSuffix(err.Error(), "connect: connection refused") {
				// 	fmt.Println("REFUSED: ", err)
				// 	handleDialFail(node, target)
				// 	i--
				// 	continue
				// } else {
				// 	fmt.Println("Dial Error: ", err)
				// 	i--
				// 	continue
				// }
				fmt.Println("Error in Gossiping: ", err)
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

func handleDialFail(node *Node, deadPeer string) {
	// node.mutex.Lock()
	node.MembersSet.SetDelete(deadPeer)
	for _, peer := range node.MembersSet.SetToArray() {
		targetPeer := strings.Split(peer, " ")
		ip := targetPeer[0]
		port := targetPeer[1]
		conn, err := net.Dial("tcp", ip+":"+port)
		deadMesg := "DEAD" + "," + deadPeer
		if err != nil {
			// if strings.HasSuffix(err.Error(), "connect: connection refused") {
			// 	fmt.Println("REFUSED: ", err)
			// 	go handleDialFail(node, peer)
			// } else {
			// 	fmt.Println("Fail Dial Error: ", err)
			// }
			continue
		} else {
			fmt.Fprintf(conn, deadMesg)
			conn.Close()
		}
	}
	// node.mutex.Unlock()
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
	// node.Members = node.MembersSet.SetToArray()
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
	serverAddr := "172.22.156.34"
	conn, err := net.Dial("tcp", serverAddr+":8888")
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		fmt.Fprintf(conn, "CONNECT "+localIP+" "+gossipPort+"\n")
	}
	return
}

func swimFailureDection(node *Node) {
	for {
		if node.MembersSet.Size() == 0 {
			time.Sleep(failureInterval)
			continue
		} else {
			NumMembers := node.MembersSet.Size()
			for i := 0; i < int(NumMembers/10); i++ {
				target := node.MembersSet.GetRandom()
				targetPeer := strings.Split(target, " ")
				ip := targetPeer[0]
				port := targetPeer[1]
				conn, err := net.Dial("tcp", ip+":"+port)
				if err != nil {
					if strings.HasSuffix(err.Error(), "connect: connection refused") {
						fmt.Println("REFUSED: ", err)
						go handleDialFail(node, target)
					} else {
						fmt.Println("Fail Dial Error: ", err)
					}
				} else {
					fmt.Fprintf(conn, "PING")
					conn.Close()
				}
			}
			time.Sleep(failureInterval)
		}
	}
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

	go swimFailureDection(node)

	// Receive message from Introduction Service
	handleServiceTCPConnection(node, introdutionConn)
}
