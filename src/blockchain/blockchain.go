package blockchain

type Block struct {
	height            int
	previousBlockHash []byte // len = 32
	transactionList   []string
}
