package main

import (
	"fmt"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"testing"
)

// SECTION: Distribution of log generation for the report

type GenerateReportLogsArgs struct {}

type GenerateReportLogsReply struct {
	Error bool
}

func (node *Node) HandleGenerateReportLogs(args *GenerateReportLogsArgs, reply *GenerateReportLogsReply) error {
	filepath := node.getLogFilepath()
	os.MkdirAll("logs", 0755)
	os.Remove(filepath)

	contentString := ""
	numLines := 1000 // configure so that we generate log files with about 60MB size

	for i := range numLines {
		// deterministically create log files
		if i % 100 < 90  { // frequent logs occur 90% of the time
			contentString += fmt.Sprintf("VM #%s: frequent log happens Frequently\n", node.PeerNumbers[node.Me])
		} else if i % 100 < 99 { // infrequent logs occur 9% of the time
			contentString += fmt.Sprintf("VM #%s: infrequent log occurs infrequently\n", node.PeerNumbers[node.Me])
		} else { // rare logs occur 1% of the time
			contentString += fmt.Sprintf("VM #%s: Rare log gonna show up rarely\n", node.PeerNumbers[node.Me])
		}
	}

	//add specific cases to each VM
	contentString += node.buildTestCase()
	content := []byte(contentString)
	err := os.WriteFile(filepath, content, 0644)

	if err != nil {
		reply.Error = true
		return err
	}

	return nil
}

// Generates report logs (IMPORTANT: use 4 nodes for the report)
func (node *Node) distributeGenerateReportLogs() { 
	var wg sync.WaitGroup
	replies := make([](GenerateReportLogsReply), len(node.Peers))
	for i, addr := range(node.Peers) {
		

		if i == node.Me { // directly call instance's own function
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				args := GenerateReportLogsArgs{}
				node.HandleGenerateReportLogs(&args, &replies[index])
			}(i)
		} else {
			wg.Add(1)
			go func(index int, peerAddr string) {
				defer wg.Done()
				client, err := rpc.Dial("tcp", peerAddr + node.Port)
				if err != nil {
					replies[index].Error = true
					return
				}
				args := GenerateReportLogsArgs{}
				err = client.Call("Node.HandleGenerateReportLogs", &args, &replies[index])

				if err != nil {
					replies[index].Error = true
					return
				}

				client.Close()
			}(i, addr)
		}
	}

	wg.Wait() // wait for all RPC and self to finish
	for i, reply := range replies {
		if reply.Error {
			fmt.Println("Error creating report logs for VM", node.Peers[i])
		}
	}
}


// SECTION: Distribution of tests for logs
type GenerateTestLogsArgs struct {
	Content string
}

type GenerateTestLogsReply struct {
	Error bool
}

func getNodeToTestLogContent() []string {
	return []string{"ALL", "ALL\nSOME", "ALL", "ALL\nSOME", "ALL", "ALL\nSOME", "ALL", "ALL\nSOME", "ALL", "ALL\nSOME"} // 10 entries for 10 VMs
}


func (node *Node) HandleGenerateTestLogs(args *GenerateTestLogsArgs, reply *GenerateTestLogsReply) error {
	filepath := node.getLogFilepath()
	os.MkdirAll("logs", 0755)
	os.Remove(filepath)

	err := os.WriteFile(filepath, []byte(args.Content), 0644)
    if err != nil {
        reply.Error = true
        return err
    }

	reply.Error = false
	return nil
}

func (node *Node) distributeGenerateTestLogs() {
	var wg sync.WaitGroup
	nodeToTestLogContent := getNodeToTestLogContent()
	replies := make([](GenerateTestLogsReply), len(node.Peers))
	for i, addr := range(node.Peers) {
		

		if i == node.Me { // directly call instance's own function
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				args := GenerateTestLogsArgs{Content: nodeToTestLogContent[i]}
				node.HandleGenerateTestLogs(&args, &replies[index])
			}(i)
		} else {
			wg.Add(1)
			go func(index int, peerAddr string) {
				defer wg.Done()
				client, err := rpc.Dial("tcp", peerAddr + node.Port)
				if err != nil {
					replies[index].Error = true
					return
				}
				args := GenerateTestLogsArgs{Content: nodeToTestLogContent[i]}
				err = client.Call("Node.HandleGenerateTestLogs", &args, &replies[index])

				if err != nil {
					replies[index].Error = true
					return
				}

				client.Close()
			}(i, addr)
		}
	}

	wg.Wait() // wait for all RPC and self to finish
	for i, reply := range replies {
		if reply.Error {
			fmt.Println("Error creating test logs for VM", node.Peers[i])
		}
	}
}

func TestSetupLogFiles(t *testing.T) {
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
	node.distributeGenerateReportLogs()
}

// SECTION: running test cases
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
			},
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

//

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
}