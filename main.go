package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type GrepArgs struct {
	Input string
}

type GrepReply struct {
	Output string
	LineCount int
	Error bool
	ErrorMsg string
}

type Node struct {
	Peers []string // VM addresses
	PeerNumbers []string // VM numbers
	Me int // the index into Peers of this VM
	Port string // hardcoded to ":12345" upon initialization
}

// Constants

func getPeers() []string {
	return []string{"fa26-cs425-1201.cs.illinois.edu", "fa26-cs425-1202.cs.illinois.edu", "fa26-cs425-1203.cs.illinois.edu", "fa26-cs425-1204.cs.illinois.edu"}
}

func getPeerNumbers() []string {
	return []string{"1201", "1202", "1203", "1204"}
}

func getPort() string {
	return ":12345"
}

// Node related functions

func (node *Node) getLogFilepath() string {
	filepath := fmt.Sprintf("logs/machine.%s.log", node.PeerNumbers[node.Me])
	return filepath
}

func (node *Node) HandleGrep (args *GrepArgs, reply *GrepReply) error {
	// implementation here to execute the command.
	// args.Input is just the raw input string that the user passed (i.e. everything after `grep` e.g. "-i -E "regex here")
	// Need to exec grep on the filepath `logs/machine.${node.PeerNumbers[node.Me]}.log`
	// example: cmd := exec.Command("sh", "-c", fullCommand); output, err = cmd.CombinedOutput()
		// don't use exec.Command("grep", ...args) directly since apparently it doesn't support all the flags and shit
	// Need to set reply.LineCount (and maybe reply.Output) and reply.Error
	output, err := node.runGrepCommand(args)
	
	// check errors
	if err != nil {
		if os.IsNotExist(err) {
			reply.Output = ""
			reply.LineCount = 0
			reply.Error = true
			reply.ErrorMsg = "Log filepath doesn't exist"
			return nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok{
			exitCode := exitErr.ExitCode()
			// exit code = 1 means that there were no pattern matches which is not an actual error
			if exitCode == 1 {
				reply.Output = ""
				reply.LineCount = 0
				reply.Error = false
				reply.ErrorMsg = ""
				return nil
			}
		}
		// if not exit code = 1, then valid error
			// potential errors: 
		// user tries to pass a filepath. once we append our own machine.i.log filepath, exec grep will return an error, so can return a generic error in this case. TODO: tell user not to specify a filepath in grep command
		// grep fails due to invalid pattern/ user input, will be caught by the exec command so can return an error
		reply.Error = true
		reply.ErrorMsg = "Syntax error in grep command. Ensure filepath is not specified and arguments are valid."
		fmt.Printf("Error executing command : %s\n ", err)
		return nil
	}
	// successful grep
	reply.Error = false
	reply.ErrorMsg = ""
	// use scanner to count lines and set reply.LineCount and reply.Output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	for scanner.Scan() {
		count++
	}
	reply.LineCount = count
	maxLines := 50 // configurable
	if count > maxLines {
        reply.Output = fmt.Sprintf("Over %d lines returned, skipping printing the output", maxLines)
    } else {
        reply.Output = string(output)
    }
	// return nil since there were no errors
	return nil
}

func (node *Node) runGrepCommand(args *GrepArgs) (output []byte, err error) {
	// TODO: this is probably wrong. spec says that i in machine.i.log should be the VM number. what does that mean?
	filepath := node.getLogFilepath()
	// extract input
	_, err = os.Stat(filepath)

	if err != nil {
		return []byte(""), err
	}

	arguments := args.Input
	// create grep command
	fullCommand := fmt.Sprintf("grep -H %s %s", arguments, filepath)
	//call grep
	cmd := exec.Command("sh", "-c", fullCommand)

	output, err = cmd.CombinedOutput()

	return output, err
}

func (node *Node) runServer() {
	rpc.Register(node)
	listener, _ := net.Listen("tcp", node.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		rpc.ServeConn(conn)
	}
}

func (node *Node) formatOutputString(replies []GrepReply) string {
	output := ""
	totalLineCount := 0
	for i, reply := range replies {
		if reply.Error {
			output += fmt.Sprintf("Error running grep for %s: %s.\n\n", node.Peers[i], reply.ErrorMsg)
		} else {
			output += fmt.Sprintf("%d Lines returned for %s:\n%s\n\n", reply.LineCount, node.Peers[i], reply.Output)
			totalLineCount += reply.LineCount
		}
	}
	output += fmt.Sprintf("\nTotal lines returned: %d\n", totalLineCount)
	return output
}

func (node *Node) runDistributeGrep(input string) []GrepReply {
	var wg sync.WaitGroup
	replies := make([](GrepReply), len(node.Peers))
	for i, addr := range(node.Peers) {
		args := GrepArgs{Input: input}

		if i == node.Me { // directly call instance's own function
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				node.HandleGrep(&args, &replies[index])
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

				err = client.Call("Node.HandleGrep", &args, &replies[index])

				if err != nil {
					replies[index].Error = true
					return
				}

				client.Close()
			}(i, addr)
		}
	}

	wg.Wait() // wait for all RPC and self grep to finish
	return replies
}

func (node *Node) distributeAndReturnGrep(input string) string {
	start := time.Now()
	replies := node.runDistributeGrep(input)
	output := node.formatOutputString(replies)
	elapsed := time.Since(start)
	output += fmt.Sprintf("Elapsed time: %s\n", elapsed)
	return output
}

func main() {
	node := new(Node)
	node.Peers = getPeers()
	node.PeerNumbers = getPeerNumbers()
	node.Me = -1
	node.Port = getPort()

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error determing VM hostname, exiting")
		return
	}

	for i, peer := range node.Peers {
		if strings.Contains(peer, hostname) {
			node.Me = i
			break
		}
	}

	go node.runServer()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your grep pattern here, excluding the filepath: grep ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		output := node.distributeAndReturnGrep(input)
		fmt.Println(output)
	}

}

// SECTION: Distribution of log generation for the report
// moved to main.go because Go will not compile tests file unless specifically run and runServer() is run here
// this can be changed in the future

type GenerateReportLogsArgs struct {}

type GenerateReportLogsReply struct {
	Error bool
}

func (node *Node) HandleGenerateReportLogs(args *GenerateReportLogsArgs, reply *GenerateReportLogsReply) error {
	filepath := node.getLogFilepath()
	os.MkdirAll("logs", 0755)
	os.Remove(filepath)

	contentString := ""

	for i := range 100 {
		// deterministically create log files
		if i < 90  { // frequent logs occur 90% of the time
			contentString += fmt.Sprintf("VM #%s: frequent log happens Frequently\n", node.PeerNumbers[node.Me])
		} else if i < 99 { // infrequent logs occur 9% of the time
			contentString += fmt.Sprintf("VM #%s: infreq log occurs infrequenttly\n", node.PeerNumbers[node.Me])
		} else { // rare logs occur 1% of the time
			contentString += fmt.Sprintf("VM #%s: Rare log gonna show up rarely\n", node.PeerNumbers[node.Me])
		}
	}

	shellCmd := fmt.Sprintf("yes '%s' | head -c 60M > %s", contentString, filepath)
    err := exec.Command("bash", "-c", shellCmd).Run()

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
//test struct
type testCase struct {
	pattern string
	countsbyVM map[int]int // machine index -> lines on machine
}