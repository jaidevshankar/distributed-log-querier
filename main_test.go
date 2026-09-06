package main

import (
	"os"
	"strings"
	"testing"
)


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

// call to setup happy path test logs
func TestSetupTestFiles(t *testing.T) {
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
	node.distributeGenerateTestLogs()
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
	// these are our general test cases representing grep for patterns on all, some, and no machines
	cases := []struct {
		name string
		pattern string
		expected int
	}{
		{"pattern_all_machines", "ALL", len(node.Peers)},
		{"pattern_some_machines", "SOME", len(node.Peers)/2},
		{"pattern_none_machines", "NONE", 0},

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

// testing grep functionality, all local to one vm

func TestHandleGrepLocal(t * testing.T) {
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
	filepath := node.getLogFilepath()
	os.MkdirAll("logs", 0755)
	// create one file in the current machine to measure our flags
	// currently, the main things we will test is flag -i for case insensitive, -E for regex check
	// also check syntax errors and user given filepaths give appropriate errors
	content := "ERROR : test case\n" + "ERROR: test case\n" +
				"error : lowercase message\n" + "EXTRA: filler\n" + "EXTRA: filler\n" +
				 "CODE99 : regex flag check\n"
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write local test log file: %v", err)
	} 
	t.Cleanup(func() { os.Remove(filepath)})
	cases := []struct {
		name string
		input string
		expectedCount int
		expectError bool
		expectedErrMsg string
	}{
		{"case_sensitive_match", "ERROR", 2, false, ""},
		{"case_insensitive_flag", "-i ERROR", 3, false, ""},
		{"extended_regex_flag", `-E "CODE[0-9]+"`, 1, false, ""},
		{"syntax_error", `-E "("`, 0, true, "Syntax error in grep command. Ensure filepath is not specified and arguments are valid."},
		{"user_supplied_filepath", "ERROR /not_real/fake/path", 0, true, "Syntax error in grep command. Ensure filepath is not specified and arguments are valid."},
	}
	for _,tc := range cases {
		t.Run(tc.name, func(t * testing.T) {
			args := GrepArgs{Input : tc.input}
			var reply GrepReply
			node.HandleGrep(&args, &reply)
			// for error cases, check if it is expected error message
			if reply.Error != tc.expectError {
				t.Fatalf("input %q: expected Error=%v, got Error=%v (msg: %s)", tc.input, tc.expectError, reply.Error, reply.ErrorMsg)
			}
			if tc.expectError {
				if reply.ErrorMsg != tc.expectedErrMsg {
					t.Errorf("input %q: expected ErrorMsg %q, got %q", tc.input, tc.expectedErrMsg, reply.ErrorMsg)
				}
				return
			}
			// check that line counts are as expected
			if reply.LineCount != tc.expectedCount {
				t.Errorf("input %q: expected %d lines, got %d", tc.input, tc.expectedCount, reply.LineCount)
			}
			//check that filepath is correct
			if !strings.Contains(reply.Output, filepath) {
				t.Errorf("input %q: expected output to include filename %q, got %q", tc.input, filepath, reply.Output)
			}
		},
		)
	}
	// check that grep correcty handles a missing log case by removing built filepath
	t.Run("missing_log_file", func(t*testing.T) {
		os.Remove(filepath)
		args := GrepArgs{Input : "ERROR"}
		var reply GrepReply
		node.HandleGrep(&args, &reply)
		if !reply.Error {
			t.Fatalf("expected Error=true for missing log file, got Error=false")
		}
		if reply.ErrorMsg != "Log filepath doesn't exist" {
			t.Errorf("expected ErrorMsg %q, got %q", "Log filepath doesn't exist", reply.ErrorMsg)
		}
	})
}

// specifically testing a node failure
func TestNodeFailure(t * testing.T) {
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
	node.distributeGenerateTestLogs()
	// specifically add a fake address  to simulate a node failure
	actualPeerCount := len(node.Peers)
	node.Peers = append(node.Peers, "fa26-cs425-9999.cs.illinois.edu") // add fake address
	fakeIdx := actualPeerCount
	replies := node.runDistributeGrep("ALL")
	// ensure that we simply receive an error statement and continue
	if !replies[fakeIdx].Error {
		t.Errorf("expected fake peer at index %d to report Error=true, got Error=false", fakeIdx)
	}
	// ensure that everything else works properly, as of now just redoing ALL test
	for i := 0; i < actualPeerCount; i++ {
		if replies[i].Error {
			t.Errorf("Error message %s", replies[i].ErrorMsg)
			t.Errorf("real peer %s unexpectedly reported an error", node.Peers[i])
			continue
		}
		if replies[i].LineCount != 1 {
			t.Errorf("real peer %s: expected 1 line for pattern \"All\", got %d", node.Peers[i], replies[i].LineCount)
		}
	}
}