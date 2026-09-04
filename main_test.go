package main

import (
	"testing"
	"os"
	"strings"
	"fmt"
	"math/rand/v2"
)

// function for running our test cases
func TestGrep(t *testing.T) {
	peers := getPeers()
	peerNumbers := getPeerNumbers()
	port := getPort()
	node := Node{Peers: peers, PeerNumbers: peerNumbers, Me: 0, Port: port}
	hostname, _ := os.Hostname()
	for i, peer := range node.Peers {
		if strings.Contains(peer, hostname) {
			node.Me = i
			break
		}
	}
	// these are our test cases representing the rare, frequent, and infrequent patterns that we are examining
	// we will manually create each case to return a certain number of lines and exist on a certain number of machines
	// we have a separate test call to create these log lines which we will call on every vm
	// this only needs to be called on this vm
	//right now, i just have placeholder line counts and we can manually create these
	cases := []struct {
		name string
		pattern string
		expected int
	}{
		{"rare_one_machine", "TEST_RARE_ONE", 3},
		{"rare_all_machines", "TEST_RARE_ALL", 2 * len(node.Peers)},
		{"rare_some_machines", "TEST_RARE_SOME", 2 * len(node.Peers)},
		{"infrequent_one_machine", "TEST_INFREQUENT_ONE",50},
		{"infrequent_all_machines", "TEST_INFREQUENT_ALL", 40*len(node.Peers)},
		{"infrequent_some_machines", "TEST_INFREQUENT_SOME", 40*len(node.Peers)},
		{"frequent_one_machine", "TEST_FREQUENT_ONE", 500},
		{"frequent_all_machines", "TEST_FREQUENT_ALL", 400*len(node.Peers)},
		{"frequent_some_machines", "TEST_FREQUENT_SOME", 400*len(node.Peers)},
		{"none_all", "TEST_NONE_ALL", 0},

	}
	// run through all tests
	for _, tc := range cases {
		t.Run(tc.name, func(t * testing.T){
			// in the previous folder, i separated runDistributeGrep so we can directly check tests
			replies := node.runDistributeGrep(tc.pattern)
			total := 0
			for i, r := range replies {
				// catching any errors
				if r.Error {
					t.Fatalf("unexpected error from %s for pattern %q", node.Peers[i], tc.pattern)
				}
				total += r.LineCount
			}
			//situation where we fail a test
			if total != tc.expected {
				t.Errorf("pattern %q: expected %d total lines, got %d", tc.pattern, tc.expected, total)
				}
			}
		)
	
	}
	
}

//test struct
type testCase struct {
	pattern string
	countsbyVM map[int]int // machine index -> lines on machine
}
//test with pattern and hardcoded count per entry
var tests = []testCase{
	{"TEST_RARE_ONE", map[int]int{0: 3, 1: 0, 2: 0, 3:0, 4:0}},
	{"TEST_RARE_ALL", map[int]int{0:2, 1:2, 2:2, 3:2, 4:2}},
	{"TEST_RARE_SOME", map[int]int{0:2,1:0,2:0,3:2,4:2}},
	{"TEST_INFREQUENT_ONE", map[int]int{0:50, 1:0, 2:0, 3:0, 4:0}},
	{ "TEST_INFREQUENT_ALL", map[int]int{0: 40, 1:40, 2:40, 3:40, 4:40}},
	{"TEST_INFREQUENT_SOME", map[int]int{0:40, 1:40, 2:0, 3:40, 4:0}},
	{"TEST_FREQUENT_ONE", map[int]int{0:500, 1:0, 2:0, 3:0, 4:0}},
	{"TEST_FREQUENT_ALL", map[int]int{0:400, 1:400, 2:400, 3:400, 4:400}},
	{"TEST_FREQUENT_SOME", map[int]int{0:400, 1:400, 2:400, 3:0, 4:0}},
	{"TEST_NONE_ALL", map[int]int{0:0, 1:0, 2:0, 3:0, 4:0}},
}

// function to build specific test cases
func (node *Node) buildTestCase() string {
	content := ""
	// iterate through every test, this is specific to each VM
	for _,entry := range tests {
		count, present := entry.countsbyVM[node.Me]
		if !present || count == 0 {
			continue
		}
		// add specifc log lines with the test identifier for as many lines as specified
		for i:= 0; i < count; i++ {
			content+= fmt.Sprintf("%s test line\n", entry.pattern)
		}
	}
	return content
}

// this is what we run in each VM to generate the tests and log lines
// right now, this is manual and we would have to call this on every vm before being able to run the tests
// there is an option to automate this with rpc requests if you want to try doing that
func TestSet(t *testing.T) {
	peers := getPeers()
	peerNumbers := getPeerNumbers()
	port := getPort()
	hostname, _ := os.Hostname()
	node := Node{Peers: peers, PeerNumbers: peerNumbers, Me: 0, Port: port}
	for i, peer := range node.Peers {
		if strings.Contains(peer, hostname) {
			node.Me = i
			break
		}
	}
	node.generateLogFiles()
}

// cleaning up testing logs
func TestDestroy(t *testing.T) {
	peers := getPeers()
	peerNumbers := getPeerNumbers()
	port := getPort()
	hostname, _ := os.Hostname()
	node := Node{Peers: peers, PeerNumbers: peerNumbers, Me: 0, Port: port}
	for i, peer := range node.Peers {
		if strings.Contains(peer, hostname) {
			node.Me = i
			break
		}
	}
	node.destroyLogFiles()
}

// Testing Helpers

func (node *Node) generateLogFiles() {
	filepath := node.getLogFilepath()
	err := os.MkdirAll("logs", 0755)

	if err != nil {
		fmt.Println("Error creating log directory")
		return
	}

	contentString := ""
	minLines := 2000
	maxLines := 4000
	numLines := minLines + rand.IntN(maxLines - minLines)

	for range numLines {
		r := rand.Float64()
		if r < 0.9 {
			contentString += fmt.Sprintf("VM %s: frequent log happens Frequently\n", node.PeerNumbers[node.Me])
		} else if r < 0.99 {
			contentString += fmt.Sprintf("VM %s: infrequent log occurs infrequently\n", node.PeerNumbers[node.Me])
		} else {
			contentString += fmt.Sprintf("VM %s: Rare log gonna show up rarely\n", node.PeerNumbers[node.Me])
		}
	}

	//add specific cases to each VM
	contentString += node.buildTestCase()
	content := []byte(contentString)
	err = os.WriteFile(filepath, content, 0644)

	if err != nil {
		fmt.Println("Error generating log file for VM", node.Peers[node.Me])
	}
}

func (node *Node) destroyLogFiles() {
	filepath := node.getLogFilepath()
	os.Remove(filepath)
	os.Remove("logs")
}