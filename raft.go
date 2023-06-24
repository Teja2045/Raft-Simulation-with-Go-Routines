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

var interrupt = false

type Node struct {
	id    int
	peers []*Node

	state    int
	term     int
	leaderID int

	value int

	electionTimeout         time.Duration
	heartbeatTimeout        time.Duration
	receiveHeartBeatTimeout time.Duration
	heartbeatCh             chan *HeartBeat
	voteRequestCh           chan *RequestVote
	voteCh                  chan bool
	timeout                 time.Duration
}

type HeartBeat struct {
	id   int
	term int
}

type RequestVote struct {
	term            int
	candidateID     int
	responseChannel chan bool
}

func (n *Node) run() {

	time.Sleep(1000 * time.Millisecond)

	fmt.Println("starting for node ", n.id)
	for {

		switch n.state {

		case FOLLOWER:

			fmt.Println(n.id, " is a FOLLOWER	")
			for {
				if n.state != FOLLOWER {
					break
				}
				select {
				case beat := <-n.heartbeatCh:
					fmt.Println(n.id, " follower received heartBeat from ", beat.id, " with term ", beat.term, "!")
					if beat.term >= n.term {
						n.leaderID = beat.id
						n.term = beat.term

						fmt.Println("node ", n.id, " follower's timer is resetting...")
						n.timeout = time.Duration(n.receiveHeartBeatTimeout)
					}

				case voteRequest := <-n.voteRequestCh:
					requestTerm := voteRequest.term
					requestCandidate := voteRequest.candidateID
					requestVotingChannel := voteRequest.responseChannel
					if requestTerm > n.term {
						fmt.Println("node ", n.id, " follower voted for ", requestCandidate)
						requestVotingChannel <- true
						n.term = voteRequest.term
						n.leaderID = requestCandidate

						fmt.Println("node ", n.id, "follower's timer is resetting due to voting...")
						n.timeout = time.Duration(n.receiveHeartBeatTimeout)

					} else {
						fmt.Println(n.id, " follower rejected ", requestCandidate)
						requestVotingChannel <- false
					}

				case <-time.After(n.timeout):
					fmt.Println(n.id, " follower is timed out!!!! its going to be a candidate")
					n.state = CANDIDATE
				}
			}

		case CANDIDATE:
			fmt.Println(n.id, " became a candidate")
			fmt.Println(n.id, " candidate's previous term: ", n.term)
			n.term++
			fmt.Println(n.id, " candidate's terms incremented term: ", n.term)
			successfulVoteChannel := make(chan RequestVote)
			allVotesChannel := make(chan bool)
			validHeartBeatCh := make(chan HeartBeat)
			exit := false
			var waitGroup sync.WaitGroup

			go func(validHeartBeatCh chan HeartBeat) {
				fmt.Println(n.id, " candidate's candidate heart beat routines is started...")
				waitGroup.Add(1)
				for {
					if n.state != CANDIDATE || exit {
						break
					}
					select {
					case heartBeat := <-n.heartbeatCh:
						if heartBeat.term >= n.term {
							fmt.Println(n.id, " candidate received valid heart beat from ", heartBeat.id, " with term ", heartBeat.term)
							validHeartBeatCh <- *heartBeat
						} else {
							fmt.Println(n.id, " received invalid heart beat from ", heartBeat.id)
						}
					default:
					}
				}
				waitGroup.Done()
				fmt.Println(n.id, " candidate's candidate hearbeat routines is stopped...")
			}(validHeartBeatCh)

			go func(allVotesChannel chan bool) {
				fmt.Println(n.id, " candidate's candidate votes counting go routine is started...")
				waitGroup.Add(1)
				votesCount := 1
				rejectCount := 0
				for {
					if votesCount > 2 {
						allVotesChannel <- true
						break
					}
					if n.state != CANDIDATE || exit {
						break
					}
					select {
					case response := <-n.voteCh:
						if response {
							votesCount++
							fmt.Println(n.id, " candidate got a vote! Voted=", votesCount, " Rejected=", rejectCount)
						} else {
							rejectCount++
							fmt.Println(n.id, " candidate got a rejection! Voted=", votesCount, " Rejected=", rejectCount)
						}
					default:
					}
				}
				waitGroup.Done()
				fmt.Println(n.id, " candidate's candidate votes counting go routine is stopped...")
			}(allVotesChannel)

			go func(successfulVoteChannel chan RequestVote) {
				fmt.Println(n.id, " candidate's voting request go routine is started...")
				waitGroup.Add(1)
				for {
					if n.state != CANDIDATE || exit {
						break
					}
					select {
					case voteRequest := <-n.voteRequestCh:
						fmt.Println(n.id, " candidate routine received a vote request from ", voteRequest.candidateID, "with term ", voteRequest.term)
						if voteRequest.term > n.term {
							successfulVoteChannel <- *voteRequest
						} else {
							voteRequest.responseChannel <- false
						}
					case <-time.After(n.electionTimeout):
						fmt.Println(n.id, ": no request yet for candidate...")
					}
				}
				waitGroup.Done()
				fmt.Println(n.id, " candidate's voting request go routine is stopped...")
			}(successfulVoteChannel)

			for _, peer := range n.peers {

				go func(peer *Node) {
					fmt.Println(n.id, " candidate is requesting", peer.id)

					peer.voteRequestCh <- &RequestVote{
						term:            n.term,
						candidateID:     n.id,
						responseChannel: n.voteCh,
					}
				}(peer)
			}

			electionTimeout := time.Duration(n.electionTimeout)

			for {
				if n.state != CANDIDATE || exit {
					break
				}
				select {
				case beat := <-validHeartBeatCh:
					fmt.Println(n.id, " candidate received a valid hear beat", beat, "! Its going to become a follower")
					n.term = beat.term
					n.leaderID = beat.id
					n.state = FOLLOWER
					n.voteCh = make(chan bool, 10)
					n.electionTimeout = resetElectionTimeout()
					n.voteRequestCh = make(chan *RequestVote, 10)
					exit = true

				case requestVote := <-successfulVoteChannel:
					fmt.Println(n.id, "candidate received a higher term voting request with term", requestVote.term, " from ", requestVote.candidateID, "! Its going to become a follower")
					n.term = requestVote.term
					n.leaderID = requestVote.candidateID
					requestVote.responseChannel <- true
					n.state = FOLLOWER
					n.voteCh = make(chan bool, 10)
					n.electionTimeout = resetElectionTimeout()
					n.voteRequestCh = make(chan *RequestVote, 10)
					exit = true

				case <-allVotesChannel:
					fmt.Println(n.id, "candidate received enough votes", "! Its going to become a Leader")
					n.leaderID = n.id
					n.state = LEADER
					n.voteCh = make(chan bool, 10)
					n.electionTimeout = resetElectionTimeout()
					n.voteRequestCh = make(chan *RequestVote, 10)

				case <-time.After(electionTimeout):
					fmt.Println(n.id, "candidate election is timed out", "! Its going to restart election")
					n.electionTimeout = resetElectionTimeout()
					n.voteCh = make(chan bool, 10)
					n.voteRequestCh = make(chan *RequestVote, 10)
					exit = true
				}

			}
			// wait until all routines are done
			waitGroup.Wait()
			fmt.Println(n.id, " is exiting candidate State")

		case LEADER:
			fmt.Println(n.id, ": is a LEADER!")
			var waitGroup sync.WaitGroup

			waitGroup.Add(1)

			go func(node *Node) {
				fmt.Println(n.id, " leader's vote request routine is started...")
				for {
					if node.state != LEADER {
						break
					}
					select {
					case voteRequest := <-node.voteRequestCh:
						requestTerm := voteRequest.term
						requestCandidate := voteRequest.candidateID
						requestVotingChannel := voteRequest.responseChannel
						fmt.Println(node.id, " leader received a vote request with term: ", requestTerm, " from ", requestCandidate)
						if requestTerm > node.term {
							node.term = requestTerm
							requestVotingChannel <- true
							node.leaderID = requestCandidate
							node.state = FOLLOWER
						}

					case heartBeat := <-n.heartbeatCh:
						if heartBeat.term > node.term {
							node.term = heartBeat.term
							node.leaderID = heartBeat.id
							node.state = FOLLOWER
						}

					default:
					}
				}
				waitGroup.Done()
				fmt.Println(n.id, " leader's vote request routine is stopped...")
			}(n)

			for {

				if interrupt {
					interrupt = false
					n.state = FOLLOWER
					fmt.Println(n.id, "leader node is malfunctioned -----------------------X!")
					time.Sleep(20000 * time.Millisecond)
					fmt.Println(n.id, "node is recovered after malfunction -------------------O!")
				}
				if n.state != LEADER {
					fmt.Println("leader ", n.id, " became a FOLLOWER!")
					break
				}

				fmt.Println(n.id, " leader is starting to send Heart Beats !")

				for _, peer := range n.peers {
					go func(peer *Node) {
						fmt.Println(n.id, " leader is sending heartbeat to follower ", peer.id)
						peer.heartbeatCh <- &HeartBeat{
							id:   n.id,
							term: n.term,
						}
					}(peer)
				}

				fmt.Println(n.id, " leader heart beat timeout started..")
				time.Sleep(n.heartbeatTimeout)
				fmt.Println(n.id, " leader heart beat timeout ended..")
			}
			waitGroup.Wait()
			fmt.Println(n.id, " leader is exiting leader state!")
		}
	}
	//fmt.Println(n.id, " is stopped for some reason")
}

func resetElectionTimeout() time.Duration {
	return time.Duration(time.Duration(rand.Intn(35000)+150) * time.Millisecond)
}

func resetReceiveHearBeatTimeout() time.Duration {
	return time.Duration(time.Duration(rand.Intn(50000)+150) * time.Millisecond)
}

func resetHeartBeatBroadCastTimeout() time.Duration {
	return time.Duration(time.Duration(rand.Intn(5000)+15) * time.Millisecond)
}

func main() {

	nodes := []*Node{}
	for i := 0; i < 5; i++ {
		node := &Node{
			id:                      i,
			state:                   FOLLOWER,
			leaderID:                0,
			term:                    0,
			value:                   -1,
			electionTimeout:         resetElectionTimeout(),
			heartbeatTimeout:        resetHeartBeatBroadCastTimeout(),
			receiveHeartBeatTimeout: resetReceiveHearBeatTimeout(),
			voteCh:                  make(chan bool, 10),
			heartbeatCh:             make(chan *HeartBeat),
			voteRequestCh:           make(chan *RequestVote),
		}
		nodes = append(nodes, node)
	}

	go func() {
		for {
			time.Sleep(50000 * time.Millisecond)
			fmt.Println("INTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT! /nINTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT! /n INTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT! INTERRUPT!")
			interrupt = true
		}
	}()

	for i, node := range nodes {
		node.peers = append(node.peers, nodes[:i]...)
		node.peers = append(node.peers, nodes[i+1:]...)
	}

	for _, node := range nodes {
		go node.run()
	}

	select {}
}
