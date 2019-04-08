package shared

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"

	"../blockchain"
)

type MergeInfo struct {
	Balance map[int]int
	Mempool StringSet
}

func MakeMergeInfo(balance map[int]int, mempool StringSet) MergeInfo {
	mergeInfo := new(MergeInfo)
	mergeInfo.Balance = balance
	mergeInfo.Mempool = mempool
	return *mergeInfo
}

// BlockBuffer : A buffer that keeps a read counter for each block
type BlockBuffer struct {
	blocks   map[*blockchain.Block]int
	maxCount int
	RWlock   sync.RWMutex
}

// NewBlockBuffer : Constructor
func NewBlockBuffer(n int) *BlockBuffer {
	buf := new(BlockBuffer)
	buf.blocks = make(map[*blockchain.Block]int)
	buf.maxCount = n
	return buf
}

// Add : Add a block to the buffer
func (buf *BlockBuffer) Add(block *blockchain.Block) {
	buf.RWlock.Lock()
	buf.blocks[block] = 0
	buf.RWlock.Unlock()
}

// GetAll : Retrieve all blocks in the buffer as an array of block pointers
func (buf *BlockBuffer) GetAll() []*blockchain.Block {
	buf.RWlock.Lock()
	defer buf.RWlock.Unlock()

	result := make([]*blockchain.Block, 0)
	for block, _ := range buf.blocks {
		buf.blocks[block]++
		result = append(result, block)
		if buf.blocks[block] > buf.maxCount {
			delete(buf.blocks, block)
		}
	}
	return result
}

// MsgBuffer : A buffer that
// 1. prioritize messages that has been read fewer times
// 2. discards a message after reading it more than N times
type MsgBuffer struct {
	buf    [][]string
	RWlock sync.RWMutex
}

// NewMsgBuffer : MsgBuffer constructor
func NewMsgBuffer(n int) *MsgBuffer {
	buf := new(MsgBuffer)
	buf.buf = make([][]string, n)
	return buf
}

// CloneMsgBuffer : MsgBuffer copy constructor
func CloneMsgBuffer(buf *MsgBuffer) *MsgBuffer {
	newBuf := new(MsgBuffer)
	newBuf.buf = append([][]string(nil), buf.buf...)
	return newBuf
}

// Add : add an element to the buffer
func (buf *MsgBuffer) Add(s string) {
	buf.RWlock.Lock()
	buf.buf[0] = append(buf.buf[0], s)
	buf.RWlock.Unlock()
}

// GetN : get N messages, prioritize messages that has been read less times,
// and update the "counter" for each message read
func (buf *MsgBuffer) GetN(n int) []string {
	buf.RWlock.Lock()
	idx := 0
	newBuf := append([][]string(nil), buf.buf...)
	res := make([]string, 0)
	for {
		if idx >= len(buf.buf) || n == 0 {
			break
		}
		l := 0
		if len(buf.buf[idx]) >= n {
			l = n
		} else {
			l = len(buf.buf[idx])
		}
		res = append(res, buf.buf[idx][:l]...)
		n -= l

		newBuf[idx] = newBuf[idx][l:]
		if idx < len(buf.buf)-1 {
			newBuf[idx+1] = append(buf.buf[idx+1], buf.buf[idx][:l]...)
		}
		idx++
	}
	buf.buf = newBuf
	buf.RWlock.Unlock()
	return res
}

// **************************************** //
// *****  Node struct defination ********* //
// *************************************** //

// ************************************* //
// *****  StringSet defination ********* //
// ************************************* //

// StringSet : Customized thread-safe set data structure for String
type StringSet struct {
	set    map[string]bool
	RWlock sync.RWMutex
}

// NewSet : Construntor for StringSet
func NewSet() *StringSet {
	s := new(StringSet)
	s.set = make(map[string]bool)
	return s
}

// SetAdd : Add method for StringSet
func (set *StringSet) SetAdd(s string) bool {
	set.RWlock.Lock()
	_, found := set.set[s]
	set.set[s] = true
	set.RWlock.Unlock()
	return !found //False if it existed already
}

// SetDelete : Delete method for StringSet
func (set *StringSet) SetDelete(s string) bool {
	set.RWlock.Lock()
	defer set.RWlock.Unlock()
	_, found := set.set[s]
	if !found {
		return false // not such element
	}
	delete(set.set, s)
	return true
}

// SetHas : Check whether String is in StringSet
func (set *StringSet) SetHas(s string) bool {
	set.RWlock.RLock()
	_, found := set.set[s]
	set.RWlock.RUnlock()
	return found
}

// SetToArray : Set to array
func (set *StringSet) SetToArray() []string {
	set.RWlock.RLock()
	defer set.RWlock.RUnlock()
	keys := make([]string, 0)
	for k := range set.set {
		keys = append(keys, k)
	}
	return keys
}

// Size : size
func (set *StringSet) Size() (size int) {
	set.RWlock.RLock()
	defer set.RWlock.RUnlock()
	size = len(set.set)
	return
}

// GetRandom : Get a random element from set
func (set *StringSet) GetRandom() string {
	set.RWlock.RLock()
	defer set.RWlock.RUnlock()
	if len(set.set) == 0 {
		return ""
	}
	i := rand.Intn(len(set.set))
	for k := range set.set {
		if i == 0 {
			return k
		}
		i--
	}
	panic("never!!")
}

// **************************************** //
// *****  Shared Helper Function  ********* //
// *************************************** //

// GetServerAddressFromNumber : Get corresponding VM address based on VM number.
func GetServerAddressFromNumber(servNum int) (serverAddress string) {
	serverAddress = "sp19-cs425-g10-"
	if servNum != 10 {
		serverAddress += fmt.Sprintf("0%d%s", servNum, ".cs.illinois.edu")
	} else {
		serverAddress += fmt.Sprintf("%d%s", servNum, ".cs.illinois.edu")
	}
	return
}

// GetNumberFromServerAddress : Get corresponding VM number based on VM address.
func GetNumberFromServerAddress(serverAddress string) int {
	s := strings.Split(serverAddress, "-")[3]
	ss := strings.Split(s, ".")[0]
	num, _ := strconv.Atoi(ss[0:])
	return num
}

// GetLocalIP returns the non loopback local IP of the host
// Reference https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

type Mempool []string

func (s Mempool) Len() int {
	return len(s)
}
func (s Mempool) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Mempool) Less(i, j int) bool {
	timeI := strings.Split(s[i], " ")[1]
	timeJ := strings.Split(s[j], " ")[1]
	floatI, _ := strconv.ParseFloat(timeI, 64)
	floatJ, _ := strconv.ParseFloat(timeJ, 64)
	return floatI < floatJ
}
