package shared

import (
	"fmt"
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

// SetHas : Check whether String is in StringSet
func (set *StringSet) SetHas(s string) bool {
	set.mux.Lock()
	_, found := set.set[s]
	set.mux.Unlock()
	return found
}

// SetToArray : Set to array
func (set *StringSet) SetToArray() []string {
	keys := make([]string, len(set.set))
	for k := range set.set {
		keys = append(keys, k)
	}
	return keys
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
