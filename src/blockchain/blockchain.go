package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"

	"../shared"
)

type MergeInfo struct {
	Balance map[int]int
	Mempool shared.StringSet
}

func MakeMergeInfo(balance map[int]int, mempool shared.StringSet) MergeInfo {
	mergeInfo := new(MergeInfo)
	mergeInfo.Balance = balance
	mergeInfo.Mempool = mempool
	return *mergeInfo
}

type Block struct {
	Height            int
	PreviousBlockHash string // len = 32
	TransactionList   []string
	PuzzleSolution    string
}

func NewBlock(height int, previousBlockHash string, transactionList []string) *Block {
	block := new(Block)
	block.Height = height
	block.PreviousBlockHash = previousBlockHash
	block.TransactionList = transactionList
	block.PuzzleSolution = ""
	return block
}

func (block *Block) GetPuzzle() string {
	oldSolution := block.PuzzleSolution
	block.PuzzleSolution = ""
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(block); err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(b.Bytes())
	block.PuzzleSolution = oldSolution
	byteArray := h.Sum(nil)
	return string(byteArray)
}

func (block *Block) GetBlockHash() string {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(block); err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(b.Bytes())
	byteArray := h.Sum(nil)
	return string(byteArray)
}
