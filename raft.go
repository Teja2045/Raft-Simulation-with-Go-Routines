package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	FOLLOWER  = 0
	CANDIDATE = 1
	LEADER    = 2
)

type Node struct {
	mu sync.Mutex

	id    int
	peers []*Node

	state       int
	term        int
	voteGranted bool
	leaderID    int

	value int

	electionTimeout         time.Duration
	heartbeatTimeout        time.Duration
	recieveHeartBeatTimeout time.Duration
	heartbeatCh             chan *HeartBeat
	voteRequestCh           chan *RequestVote
	voteCh                  chan bool
	timeout                 time.Duration
}

type HeartBeat struct {
	id  int
	val int
}

type RequestVote struct {
	term            int
	candidateID     int
	responseChannel chan bool
}

//ch := make(chan int)

func (n *Node) run() {

	fmt.Println("starting for node ", n.id)
	for {

		fmt.Println(n.id, " has a value of ", n.value)

		switch n.state {

		case FOLLOWER:
			fmt.Println(n.id, " is a follower")

			select {
			case beat := <-n.heartbeatCh:

				fmt.Println(n.id, " recieved beat from..", beat.id)
				n.leaderID = beat.id
				n.value = beat.val
				fmt.Println("value of node ", n.id, " is ", n.value)
				fmt.Println("node ", n.id, " timer is resetting...")
				n.timeout = time.Duration(n.recieveHeartBeatTimeout)

			case req := <-n.voteRequestCh:
				fmt.Println(n.id, " recieved a vote request..")
				if req.term > n.term && !n.voteGranted {
					req.responseChannel <- true
				} else {
					req.responseChannel <- false
				}

			case <-time.After(n.timeout):
				n.mu.Lock()
				n.state = CANDIDATE
				n.mu.Unlock()

			case req := <-n.voteRequestCh:
				if req.term > n.term {
					req.responseChannel <- true
				} else {
					req.responseChannel <- false
				}
			}

		case CANDIDATE:
			fmt.Println(n.id, " became a candidate")
			n.mu.Lock()
			n.term++
			//n.voteGranted = 1
			n.mu.Unlock()

			for _, peer := range n.peers {

				go func(peer *Node) {
					fmt.Println(n.id, "is requesting", peer.id)

					peer.voteRequestCh <- &RequestVote{
						term:            n.term,
						candidateID:     n.id,
						responseChannel: n.voteCh,
					}
				}(peer)
			}

			// wait for votes
			votesReceived := 1
			for votesReceived < len(n.peers)/2+1 {
				select {
				case <-n.voteCh:
					votesReceived++

				case beat := <-n.heartbeatCh:
					// received heartbeat from leader, become follower
					//fmt.Println(n.id)
					n.mu.Lock()
					n.leaderID = beat.id
					n.state = FOLLOWER
					n.mu.Unlock()
					goto end

				case req := <-n.voteRequestCh:
					if req.term > n.term {
						req.responseChannel <- true
					} else {
						req.responseChannel <- false
					}
				}
			}
		end:

			// become leader
			n.mu.Lock()
			n.state = LEADER
			n.leaderID = n.id
			n.mu.Unlock()
		case LEADER:
			// send heartbeat to followers
			n.mu.Lock()
			fmt.Println(n.id, "is leader")
			fmt.Println(n.id, "is ressetting all other peers so that there wont be multiple leaders...")
			for _, peer := range n.peers {
				go func(peer *Node) {
					peer.mu.Lock()
					peer.state = FOLLOWER
					fmt.Println(n.id, " leader is sending heartbeat to follower ", peer.id)
					peer.heartbeatCh <- &HeartBeat{
						id:  n.id,
						val: 6,
					}
					peer.mu.Unlock()
				}(peer)
			}

			time.Sleep(n.heartbeatTimeout)
			n.mu.Unlock()
		}
	}
	fmt.Println(n.id, " is stopped for some reason")
}

func main() {
	// initialize nodes
	nodes := []*Node{}
	for i := 0; i < 5; i++ {
		node := &Node{
			id:                      i,
			state:                   FOLLOWER,
			leaderID:                0,
			term:                    0,
			value:                   -1,
			electionTimeout:         time.Duration(rand.Intn(150)+150) * time.Millisecond,
			heartbeatTimeout:        50 * time.Millisecond,
			recieveHeartBeatTimeout: 500 * time.Millisecond,
			voteGranted:             false,
			voteRequestCh:           make(chan *RequestVote),
			voteCh:                  make(chan bool),
		}
		nodes = append(nodes, node)
	}
	nodes[0].state = LEADER

	// set up peer relationships
	for i, node := range nodes {
		node.peers = append(node.peers, nodes[:i]...)
		node.peers = append(node.peers, nodes[i+1:]...)
	}

	// run nodes in separate goroutines
	for _, node := range nodes {
		go node.run(ch)
	}

	// go func() {
	// 	reader := bufio.NewReader(os.Stdin)
	// 	for {
	// 		fmt.Print("Enter a value: ")
	// 		text, _ := reader.ReadString('\n')

	// 		// Convert the input string to an integer
	// 		value, err := strconv.Atoi(text)
	// 		if err != nil {
	// 			fmt.Println("Invalid input. Please enter an integer.")
	// 			continue
	// 		}

	// 	// Send the value to the channel
	// 		ch <- value
	// 	}
	// }(ch)

	select {}
}
