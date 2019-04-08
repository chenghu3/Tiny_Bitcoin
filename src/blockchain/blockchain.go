package blockchain

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"../shared"
)

const batchSize = 300

// VerifyBlock : check the integrity of the recieved block
func VerifyBlock(node *shared.Node, block *shared.Block) bool {
	// TODO
	puzzle := block.GetPuzzle()
	solution := block.PuzzleSolution
	fmt.Fprintf(*node.ServiceConn, "VERIFY "+puzzle+" "+solution+"\n")
	verifyChan := make(chan bool)
	node.VerifyChannelMap[puzzle] = verifyChan
	ok := <-verifyChan
	return ok
}

// updateBlockChain : Local update BlockChain, Mempool, Balance, potential handle Switch Chain
func updateBlockChain(node *shared.Node, block *shared.Block, isLocal bool) {
	localHeight := len(node.BlockChain) - 1
	//  if is a local solved block, no need to consider Switch Chain
	if isLocal || (block.Height == localHeight+1 && node.BlockChain[localHeight].GetBlockHash() == block.PreviousBlockHash) {
		// Update BlockChain
		node.BlockChain = append(node.BlockChain, *block)
		// Update Mempool
		for _, transaction := range block.TransactionList {
			node.Mempool.SetDelete(transaction)
		}
		updateBalance(node, block)
	} else {
		// Switch Chain
		// TODO: Ask for previous blocks, mempool, balance
	}
}

// updateBalance : update account balance, reject any transactions that cause the account balance go negative
func updateBalance(node *shared.Node, block *shared.Block) {
	for _, transaction := range block.TransactionList {
		arr := strings.Split(transaction, " ")
		srcAccount, _ := strconv.Atoi(arr[3])
		destAccount, _ := strconv.Atoi(arr[4])
		amount, _ := strconv.Atoi(arr[5])

		newBalance := node.Balance[srcAccount] - amount
		if newBalance >= 0 || srcAccount == 0 {
			node.Balance[srcAccount] = newBalance
			node.Balance[destAccount] += amount
		}
	}
}

// SendBlock : One time send block
func SendBlock(node *shared.Node, block *shared.Block) {
	target := node.MembersSet.GetRandom()
	targetPeer := strings.Split(target, " ")
	ip := targetPeer[0]
	port := targetPeer[1]
	conn, _ := net.Dial("tcp", ip+":"+port)
	encoder := gob.NewEncoder(conn)
	// send gossipMesg to peer
	gossipMesg := "BLOCK\n"
	fmt.Fprintf(conn, gossipMesg)
	encoder.Encode(*block)
}

// ReadBlock : Read block use gob
func ReadBlock(reader *bufio.Reader) *shared.Block {
	decoder := gob.NewDecoder(reader)
	block := &shared.Block{}
	err := decoder.Decode(block)
	if err != nil {
		fmt.Println("Read Block Error:", err)
	}
	fmt.Println("Decoded Block with previous hash: " + block.PreviousBlockHash)
	return block
}

// RecievedBlockHandler : Handle recieved Block through gossip protocal
func RecievedBlockHandler(node *shared.Node, block *shared.Block) {
	node.RWlock.RLock()
	if block.Height > node.CurrHeight {
		node.RWlock.RUnlock()
		node.RWlock.Lock()
		node.CurrHeight++
		node.RWlock.Unlock()
		isVerifySuccess := VerifyBlock(node, block)
		if !isVerifySuccess {
			node.RWlock.Lock()
			node.CurrHeight--
			node.RWlock.Unlock()
			fmt.Println("Verify Failed!")
		} else {
			// 1. TODO gossip block
			go SendBlock(node, block)
			// 2. update blockchain, mempool acoount
			go updateBlockChain(node, block, false)
		}
	} else {
		node.RWlock.RUnlock()
	}
}

// SwimBatchPuzzleGenerator : PuzzleGenerator called in SWIM Ping function
func SwimBatchPuzzleGenerator(node *shared.Node) {
	node.RWlock.RLock()
	if node.NewMsgCount >= batchSize {
		node.RWlock.RUnlock()
		solve(node)
		node.NewMsgCount = 0
	} else {
		node.RWlock.RUnlock()
	}
}

// solve : compute puzzle hash and send it to service
func solve(node *shared.Node) {
	height := len(node.BlockChain)
	var previousBlockHash string
	if height == 0 {
		previousBlockHash = ""
	} else {
		previousBlockHash = node.BlockChain[height-1].GetBlockHash()
	}
	sortedMempool := node.Mempool.SetToArray()
	sort.Sort(shared.Mempool(sortedMempool))
	sortedMempool = sortedMempool[:2000]
	block := shared.NewBlock(len(node.BlockChain), previousBlockHash, sortedMempool)
	node.TentativeBlock = block
	puzzle := block.GetPuzzle()
	fmt.Println("Sending SOLVE...")
	fmt.Fprintf(*node.ServiceConn, "SOLVE "+puzzle+"\n")
}

// PuzzleSolvedHandler : handle TentativeBlock once recieve SOLVED from service
func PuzzleSolvedHandler(node *shared.Node, rawMsg string) {
	fmt.Println("SOLUTION RECEIVED: " + rawMsg)
	arr := strings.Split(rawMsg, " ")
	solution := arr[2]
	node.TentativeBlock.PuzzleSolution = solution
	// Update BlockChain and Mempool
	// TODO
	updateBlockChain(node, node.TentativeBlock, true)
	// Gossip Block
	// TODO
}
