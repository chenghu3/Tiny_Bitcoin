package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type Block struct {
	height            int
	previousBlockHash string // len = 32
	transactionList   []string
	puzzleSolution    string
}

func (block *Block) getPuzzle() string {
	oldSolution := block.puzzleSolution
	block.puzzleSolution = ""
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(block); err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(b.Bytes())
	block.puzzleSolution = oldSolution
	byteArray := h.Sum(nil)
	return string(byteArray)
}

func (block *Block) getBlockHash() string {
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
