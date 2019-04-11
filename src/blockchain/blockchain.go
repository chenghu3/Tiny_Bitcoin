package blockchain

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"../shared"
)

const batchSize = 300

// **************************************** //
// *****  Recieve Block Handle *********** //
// *************************************** //

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
		oldCurrHeight := node.CurrHeight
		node.CurrHeight = block.Height
		node.RWlock.Unlock()
		isVerifySuccess := verifyBlock(node, block)
		if !isVerifySuccess {
			// This should never happen
			node.RWlock.Lock()
			node.CurrHeight = oldCurrHeight
			node.RWlock.Unlock()
			fmt.Println("Verify Failed!")
		} else {
			node.BlockBuffer.Add(block)
			// 2. update blockchain, mempool acoount
			go updateBlockChain(node, block, false)
		}
	} else {
		node.RWlock.RUnlock()
	}
}

// verifyBlock : check the integrity of the recieved block
func verifyBlock(node *shared.Node, block *shared.Block) bool {
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
	// start with 1 now
	node.RWlock.Lock()
	defer node.RWlock.Unlock()

	if !isLocal {
		fmt.Println("RECEIVENEWBLOCK " + time.Now().Format("2006-01-02 15:04:05.000000") + " " + strconv.Itoa(block.Height) + " " + block.PreviousBlockHash + " " + block.SourceIP)
	}

	localHeight := len(node.BlockChain)
	//  if is a local solved block, no need to consider Switch Chain
	if isLocal || localHeight == 0 || (block.Height == localHeight+1 && node.BlockChain[localHeight-1].GetBlockHash() == block.PreviousBlockHash) {
		fmt.Println("UPDATING BLOCK CHAIN")
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
		fmt.Println("SWITCH CHAIN")
		requestMergeInfo(node, block)
		remoteAdrr := block.SourceIP
		conn, err := net.Dial("tcp", remoteAdrr)
		if err != nil {
			fmt.Println("Dial error in requestBlock.")
			log.Fatal("dialing:", err)
		}
		if block.Height > localHeight+1 {
			for i := localHeight + 1; i < block.Height; i++ {
				targetBlock := requestBlock(conn, i)
				node.BlockChain = append(node.BlockChain, *targetBlock)
			}
		}
		node.BlockChain = append(node.BlockChain, *block)
		currIdx := localHeight
		for {
			if currIdx >= 1 && node.BlockChain[currIdx-1].GetBlockHash() != node.BlockChain[currIdx].PreviousBlockHash {
				newBlock := requestBlock(conn, node.BlockChain[currIdx-1].Height)
				node.BlockChain[currIdx-1] = *newBlock
				currIdx--
			} else {
				break
			}
		}
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
	fmt.Print("BALANCEUPDATE " + node.Port + " SOURCE " + block.SourceIP + " HEIGHT " + strconv.Itoa(block.Height) + " NEWBALANCE ")
	fmt.Println(node.Balance)
}

// **************************************** //
// *******  Swith Chain Handle *********** //
// *************************************** //

// HandleMergeInfoRequst : Gossip server handle mergeInfor request
func HandleMergeInfoRequst(node *shared.Node, conn net.Conn) {
	encoder := gob.NewEncoder(conn)
	balance := node.Balance
	pool := node.Mempool.SetToArray()
	merge := shared.MakeMergeInfo(balance, pool)
	encoder.Encode(merge)
}

// HandleBlockRequst : Gossip server handle Block request
func HandleBlockRequst(node *shared.Node, conn net.Conn, requestMesg string) {
	requestHeight, _ := strconv.Atoi(strings.Split(requestMesg, " ")[1])
	if requestHeight == 0 {
		fmt.Println("requestHeight is 0, consider check")
	}
	encoder := gob.NewEncoder(conn)
	targetBlock := node.BlockChain[requestHeight-1]
	encoder.Encode(targetBlock)
}

func requestMergeInfo(node *shared.Node, block *shared.Block) {
	remoteAdrr := block.SourceIP
	conn, err := net.Dial("tcp", remoteAdrr)
	if err != nil {
		fmt.Println("Dial error in requestMergeInfo.")
		log.Fatal("dialing:", err)
	}
	// Request header
	fmt.Fprintf(conn, "RequestMergeInfo\n")
	// Wait for peer response
	dec := gob.NewDecoder(conn)
	m := &shared.MergeInfo{}
	dec.Decode(m)
	if len(m.Balance) == 0 {
		fmt.Println("Mergeinfo request Fail!!")
	}
	// lock node when update
	mempoolSet := shared.ArrayToSet(m.Mempool)
	node.RWlock.Lock()
	node.Balance = m.Balance
	node.Mempool = mempoolSet
	node.RWlock.Unlock()
	conn.Close()
}

func requestBlock(conn net.Conn, height int) *shared.Block {
	// Request header
	fmt.Fprintf(conn, "RequestBlock "+strconv.Itoa(height)+" \n")
	// Wait for peer response
	dec := gob.NewDecoder(conn)
	b := &shared.Block{}
	dec.Decode(b)
	if len(b.TransactionList) == 0 {
		fmt.Println("Block request Fail!!")
	}
	return b
}

// **************************************** //
// *******  Slove Block Handle *********** //
// *************************************** //

// SwimBatchPuzzleGenerator : PuzzleGenerator called in SWIM Ping function ???
func SwimBatchPuzzleGenerator(node *shared.Node) {
	node.RWlock.Lock()
	if node.NewMsgCount >= batchSize {
		node.NewMsgCount = 0
		node.RWlock.Unlock()
		solve(node)
	} else {
		node.RWlock.Unlock()
	}
}

// solve : compute puzzle hash and send it to service
func solve(node *shared.Node) {
	node.RWlock.RLock()
	height := node.CurrHeight
	node.RWlock.RUnlock()
	var previousBlockHash string
	if height == 0 {
		previousBlockHash = ""
	} else {
		previousBlockHash = node.BlockChain[height-1].GetBlockHash()
	}
	node.RWlock.RLock()
	sortedMempool := node.Mempool.SetToArray()
	node.RWlock.RUnlock()
	sort.Sort(shared.Mempool(sortedMempool))
	if len(sortedMempool) > 2000 {
		sortedMempool = sortedMempool[:2000]
	}
	localIP := shared.GetLocalIP()
	SourceIP := localIP + ":" + node.Port
	block := shared.NewBlock(height+1, previousBlockHash, sortedMempool, SourceIP)
	node.TentativeBlock = block
	puzzle := block.GetPuzzle()
	fmt.Println("Sending SOLVE: " + "SOLVE " + puzzle + "\n")

	fmt.Fprintf(*node.ServiceConn, "SOLVE "+puzzle+"\n")
}

// PuzzleSolvedHandler : handle TentativeBlock once recieve SOLVED from service
func PuzzleSolvedHandler(node *shared.Node, rawMsg string) {
	fmt.Println("SOLUTION RECEIVED: " + rawMsg)
	arr := strings.Split(rawMsg, " ")
	solution := arr[2]
	node.TentativeBlock.PuzzleSolution = solution
	node.RWlock.Lock()
	node.CurrHeight++
	node.RWlock.Unlock()
	// Update BlockChain and Mempool
	updateBlockChain(node, node.TentativeBlock, true)
	// Gossip Block
	node.BlockBuffer.Add(node.TentativeBlock)

	fmt.Println("NEWBLOCK " + time.Now().Format("2006-01-02 15:04:05.000000") + " " + strconv.Itoa(node.TentativeBlock.Height) + " " + node.TentativeBlock.PreviousBlockHash + " " + node.TentativeBlock.SourceIP)
	for _, transaction := range node.TentativeBlock.TransactionList {
		fmt.Println("BLOCKTRANSACTION " + transaction)
	}
	// fmt.Println("BLOCKTRANSACTION HEAD " + node.TentativeBlock.TransactionList[0])
	// fmt.Println("BLOCKTRANSACTION TAIL " + node.TentativeBlock.TransactionList[len(node.TentativeBlock.TransactionList)-1])
}

// **************************************** //
// ****** Block Gossip functions ********* //
// *************************************** //

// SendBlock : One time send block
func SendBlock(node *shared.Node, conn net.Conn, block *shared.Block) {
	// target := node.MembersSet.GetRandom()
	// targetPeer := strings.Split(target, " ")
	// ip := targetPeer[0]
	// port := targetPeer[1]
	// conn, _ := net.Dial("tcp", ip+":"+port)
	encoder := gob.NewEncoder(conn)
	// send gossipMesg to peer
	gossipMesg := "BLOCK\n"
	fmt.Fprintf(conn, gossipMesg)
	encoder.Encode(*block)
}
