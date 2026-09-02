package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
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
	// example: cmd := exec.Command("sh", "-c", fullCommand); output, err = cmd.CombinedOutput()
		// don't use exec.Command("grep", ...args) directly since apparently it doesn't support all the flags and shit
	// Need to set reply.LineCount (and maybe reply.Output) and reply.Error
	

	// potential errors: 
// user tries to pass a filepath. once we append our own machine.i.log filepath, exec grep will return an error, so can return a generic error in this case. TODO: tell user not to specify a filepath in grep command
// grep fails due to invalid pattern/ user input, will be caught by the exec command so can return an error
// no lines matched, which will technically return an error when calling cmd.CombinedOutput(). the exit status will be 1, in which case we don't want to set reply.Error to true. we just want to set reply.Output = "" and reply.LineCount = 0

// otherwise return nil since there were no errors
	return nil
}