package gossip

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"../blockchain"
	"../shared"
)

const gossipInterval time.Duration = 500 * time.Millisecond
const pingInterval time.Duration = 300 * time.Millisecond
const batchSize = 300

var batchRwlock sync.RWMutex
var heightRwlock sync.RWMutex

// HandleServiceTCPConnection : handle messge revieved from introduction service
func HandleServiceTCPConnection(node *shared.Node, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		rawMsg, err := reader.ReadString('\n')
		logBandwithInfo("Recieve", len(rawMsg))
		rawMsg = strings.Trim(rawMsg, "\n")
		if err == io.EOF {
			fmt.Println("Server offline")
			break
		}

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
			node.Mempool.SetAdd(rawMsg)
			batchRwlock.Lock()
			node.NewMsgCount++
			batchRwlock.Unlock()
			logWithTimestamp(rawMsg)
			// go sendGossipingMsg(node, "TRANSACTION", 0, rawMsg)
		} else if strings.HasPrefix(rawMsg, "DIE") || strings.HasPrefix(rawMsg, "QUIT") {
			os.Exit(1)
		} else if strings.HasPrefix(rawMsg, "SOLVED") {
			fmt.Println("SOLUTION RECEIVED: " + rawMsg)
			arr := strings.Split(rawMsg, " ")
			solution := arr[2]
			node.TentativeBlock.PuzzleSolution = solution
			// Update BlockChain and Mempool
			// TODO

			// Gossip Block
			// TODO
		} else if strings.HasPrefix(rawMsg, "VERIFY") {
			arr := strings.Split(rawMsg, " ")
			response := arr[1]
			blockHash := arr[2]
			node.VerifyChannelMap[blockHash] <- response == "OK"
		} else {
			fmt.Println("Unknown message format.")
		}

	}
}

// StartGossipServer : set tcp gossip server
func StartGossipServer(node *shared.Node, port string) {
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
func handleGossipTCPConnection(node *shared.Node, conn net.Conn) {
	defer conn.Close()

	addr := strings.Split(conn.RemoteAddr().String(), ":")[0]

	reader := bufio.NewReader(conn)

	stop := false
	for !stop {
		gossipRawMsg, err := reader.ReadString('\n')
		logBandwithInfo("Recieve", len(gossipRawMsg))
		stop = err == io.EOF
		gossipRawMsg = strings.Trim(gossipRawMsg, "\n")

		if strings.HasPrefix(gossipRawMsg, "HELLO") {
			addrPort := addr + " " + strings.Split(gossipRawMsg, " ")[1]
			node.MembersSet.SetAdd(addrPort)
			memberString := strings.Join(node.MembersSet.SetToArray(), ",") + "\n"
			logBandwithInfo("Send", len(memberString))
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
				node.Mempool.SetAdd(rawMsg)
				batchRwlock.Lock()
				node.NewMsgCount++
				batchRwlock.Unlock()
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
		} else if strings.HasPrefix(gossipRawMsg, "BLOCK") {
			block := blockchain.ReadBlock(reader)
			heightRwlock.RLock()
			if block.Height > node.CurrHeight {
				heightRwlock.RUnlock()
				heightRwlock.Lock()
				node.CurrHeight++
				heightRwlock.Unlock()
				isVerifySuccess := blockchain.VerifyBlock(node, block)
				if !isVerifySuccess {
					heightRwlock.Lock()
					node.CurrHeight--
					heightRwlock.Unlock()
					fmt.Println("Verify Failed!")
				} else {
					// 1. TODO gossip block
					go blockchain.SendBlock(node, block)
					// 2. update blockchain, mempool acoount
				}
			} else {
				heightRwlock.RUnlock()
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
func sendGossipingMsg(node *shared.Node, header string, round int, mesg string) {
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
				// try another peer
				i--
			} else {
				logBandwithInfo("Send", len(gossipMesg))
				fmt.Fprintf(conn, gossipMesg)
				conn.Close()
			}
		}
		round++
		time.Sleep(gossipInterval)
	}
}

// Ping : SWIM style dissemination of membership updates
func Ping(node *shared.Node) {
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
			logBandwithInfo("Send", len(swimMsg))
			fmt.Fprintf(conn, swimMsg)
			fmt.Print("SWIM SENT " + swimMsg)
			transactionsMsg := strings.Join(node.TransactionBuffer.GetN(10000), "\n") + "\n"
			logBandwithInfo("Send", len(transactionsMsg))
			fmt.Fprintf(conn, transactionsMsg)

			batchRwlock.Lock()
			if node.NewMsgCount >= batchSize {
				blockchain.Solve(node)
				node.NewMsgCount = 0
			}
			batchRwlock.Unlock()

			conn.Close()
		}
	}
}

// Notify existing nodes known from INTRODUCE message this node has join the P2P network.
func joinP2P(node *shared.Node, addr string, port string) {
	conn, err := net.Dial("tcp", addr+":"+port)
	defer conn.Close()
	// send hello to peer
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		logBandwithInfo("Send", len("HELLO "+node.Port+"\n"))
		fmt.Fprintf(conn, "HELLO "+node.Port+"\n")
	}
	// receive membershiplist back
	reader := bufio.NewReader(conn)
	membershiplist, err := reader.ReadString('\n')
	logBandwithInfo("Recieve", len(membershiplist))
	if err == io.EOF {
		hostname, _ := net.LookupAddr(addr)
		remotehost := hostname[0]
		machineID := shared.GetNumberFromServerAddress(remotehost)
		fmt.Println("Peer from VM" + strconv.Itoa(machineID) + " in port" + port + " went offline.")
		return
	}
	updateMembershiplist(node, membershiplist)
}

func updateMembershiplist(node *shared.Node, membershiplist string) {
	m := strings.Trim(membershiplist, "\n")
	members := strings.Split(m, ",")
	for _, member := range members {
		if !node.MembersSet.SetHas(member) {
			node.MembersSet.SetAdd(member)
		}
	}
}

// ConnectToIntroduction TODO: change introduction server to known server
func ConnectToIntroduction(node *shared.Node, gossipPort string) (conn net.Conn) {
	// Get local IP address
	localIP := shared.GetLocalIP()
	if localIP == "" {
		fmt.Println("Error: cannot find local IP.")
		return
	}
	node.MembersSet.SetAdd(localIP + " " + gossipPort)
	// Connect to Introduction Service
	serverAddr := "172.22.156.34"
	conn, err := net.Dial("tcp", serverAddr+":8888")
	node.ServiceConn = &conn
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
	} else {
		logBandwithInfo("Send", len("CONNECT "+localIP+" "+gossipPort+"\n"))
		fmt.Fprintf(conn, "CONNECT "+localIP+" "+gossipPort+"\n")
	}
	return
}

func logWithTimestamp(rawMsg string) {
	fmt.Println("LOG " + time.Now().Format("2006-01-02 15:04:05.000000") + " " + rawMsg)
}

func logBandwithInfo(direction string, byteCount int) {
	fmt.Println("Bandwith " + direction + " " + time.Now().Format("15:04:05") + " " + strconv.Itoa(byteCount))
}
