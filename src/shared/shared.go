package shared

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
)

// **************************************** //
// *****  Node struct defination ********* //
// *************************************** //

// ************************************* //
// *****  StringSet defination ********* //
// ************************************* //

// StringSet : Customized set data structure for String
type StringSet struct {
	set map[string]bool
	mux sync.Mutex
}

// NewSet : Construntor for StringSet
func NewSet() *StringSet {
	s := new(StringSet)
	s.set = make(map[string]bool)
	return s
}

// SetAdd : Add method for StringSet
func (set *StringSet) SetAdd(s string) bool {
	_, found := set.set[s]
	set.mux.Lock()
	set.set[s] = true
	set.mux.Unlock()
	return !found //False if it existed already
}

// SetDelete : Delete method for StringSet
func (set *StringSet) SetDelete(s string) bool {
	_, found := set.set[s]
	if !found {
		return false // not such element
	}
	set.mux.Lock()
	delete(set.set, s)
	set.mux.Unlock()
	return true
}

// SetHas : Check whether String is in StringSet
func (set *StringSet) SetHas(s string) bool {
	set.mux.Lock()
	_, found := set.set[s]
	set.mux.Unlock()
	return found
}

// SetToArray : Set to array
func (set *StringSet) SetToArray() []string {
	keys := make([]string, 0)
	for k := range set.set {
		keys = append(keys, k)
	}
	return keys
}

// Size : size
func (set *StringSet) Size() int {
	return len(set.set)
}

// GetRandom : Get a random element from set
func (set *StringSet) GetRandom() string {
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

func ParseMessage(msg string) (string, string, string) {
	params := strings.Split(msg, " ")
	return params[0], params[1], params[2]
}
