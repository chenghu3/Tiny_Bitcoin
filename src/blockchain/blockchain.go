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

// NewVerifiedBlockHandler : Local update BlockChain, Mempool, Balance, potential handle Switch Chain
func NewVerifiedBlockHandler(node *shared.Node, block *shared.Block) {
	localHeight := len(node.BlockChain) - 1

	if block.Height == localHeight+1 && node.BlockChain[localHeight].GetBlockHash() == block.PreviousBlockHash {
		// Update BlockChain
		node.BlockChain = append(node.BlockChain, *block)
		// Update Mempool
		for _, transaction := range block.TransactionList {
			node.Mempool.SetDelete(transaction)
		}
		UpdateBalance(node, block)
	} else {
		// Switch Chain
		// TODO: Ask for previous blocks, mempool, balance
	}
}

// UpdateBalance : update account balance, reject any transactions that cause the account balance go negative
func UpdateBalance(node *shared.Node, block *shared.Block) {
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

// Solve : compute puzzle hash and send it to service
func Solve(node *shared.Node) {
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
