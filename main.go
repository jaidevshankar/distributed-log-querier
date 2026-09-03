package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
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
	Port string // hardcoded to ":12345" upon initialization
}

func (node *Node) HandleGrep (args *GrepArgs, reply *GrepReply) error {
	// implementation here to execute the command.
	// args.Input is just the raw input string that the user passed (i.e. everything after `grep` e.g. "-i -E "regex here")
	// Need to exec grep on the filepath `logs/machine.${node.Me}.log`
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

func (node *Node) runGrep(args *GrepArgs, reply *GrepReply)  {
	// os.exec(...)
}

// Return an error only if grep is misformatted or specifies a filepath
func (node *Node) distributeAndReturnGrep(input string) (string, error) {
	var wg sync.WaitGroup
	replies := make([](GrepReply), len(node.Peers))
	for i, addr := range(node.Peers) {
		args := GrepArgs{}
		args.Input = input

		if i == node.Me {
			wg.Go(func() {
				node.runGrep(&args, &replies[i]) 
			})
		} else {
			wg.Go(func() {
				// send RPC
				client, err := rpc.Dial("tcp", addr + node.Port)
				if err != nil {
					replies[i].Error = true
					return
				}

				err = client.Call("Node.HandleGrep", &args, &replies[i])

				if err != nil {
					replies[i].Error = true
					return
				}

				client.Close()

			})
		}

		

	}
	wg.Wait()
	return "", nil
}

func main() {
	node := new(Node)
	node.Peers = []string{"fa26-cs425-01.cs.illinois.edu"}
	node.Me = -1 // TODO: update to match VM name to index into peers
	node.Port = ":12345"

	go node.runServer()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your grep pattern here, excluding the filepath: grep ")
		input, _ := reader.ReadString('\n')
		output, err := node.distributeAndReturnGrep(input)

		if err != nil {
			fmt.Println("Error running grep: ", err, ". Ensure that your syntax is correct and no filepath is specified.")
		} else {
			fmt.Println(output)
		}
		
	}

}