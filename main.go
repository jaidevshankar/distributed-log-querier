package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"bytes"
	"os/exec"
)

type GrepArgs struct {
	Input string
}

type GrepReply struct {
	Output string
	LineCount int
	Error bool
}

type Node struct {
	Peers []string // VM addresses
	Me int // the index into Nodes of this VM
	Port string // hardcoded to 12345 upon initialization
}

func (node *Node) HandleGrep (args *GrepArgs, reply *GrepReply) error {
	// implementation here to execute the command.
	// args.Input is just the raw input string that the user passed (i.e. everything after `grep` e.g. "-i -E "regex here")
	// Need to exec grep on the filepath `logs/machine.${dlq.Me}.log`
	// Need to set reply.LineCount (and maybe reply.Output) and reply.Error
	filepath := fmt.Sprintf("logs/machine.%d.log", node.Me)
	// extract input
	arguments := args.Input
	// create grep command
	fullCommand := fmt.Sprintf("grep %s %s", arguments, filepath)
	//call grep
	cmd := exec.Command("sh", "-c", fullCommand)
	output, err := cmd.CombinedOutput()
	// check errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok{
			exitCode := exitErr.ExitCode()
			// exit code = 1 means that there were no pattern matches which is not an actual error
			if exitCode == 1 {
				reply.Output = ""
				reply.LineCount = 0
				reply.Error = false
				return nil
			}
		}
		// if not exit code = 1, then valid error
			// potential errors: 
		// user tries to pass a filepath. once we append our own machine.i.log filepath, exec grep will return an error, so can return a generic error in this case. TODO: tell user not to specify a filepath in grep command
		// grep fails due to invalid pattern/ user input, will be caught by the exec command so can return an error
		reply.Error = true
		fmt.Printf("Error executing command : %s\n ", err)
		return nil
	}
	// successful grep
	reply.Error = false
	// use scanner to count lines and set reply.LineCount and reply.Output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	for scanner.Scan() {
		count++
	}
	reply.LineCount = count
	reply.Output = string(output)
// return nil since there were no errors
	return nil
}