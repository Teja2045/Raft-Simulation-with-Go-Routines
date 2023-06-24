 It is a simple simulation of raft protocol leader selection algorithm with Go routines and channels
    
## Installation:
#### Pre-requsites :- 
- Install Golang (v1.19)
- Install  Git

#### Run this command in terminal :-
    git clone https://github.com/Teja2045/Raft-Simulation-with-Go-Routines
    
## Run:
#### Run this command to run the simulation :-
    go run raft.go
    
## Features:
- Leader Election
- Failure Detection and Recovery
- Single Leader at a time
- simple state machine
- Data communication via go Channels

## Implentaion Details

- Considered 5 nodes for simplicity (increasing the number of nodes won't cause any problem).
- At the start, Each node is initialized with ID, Term=0, different timeouts like Election timeout, Heart Beat timout and different channels like voteRequest channel, HeartBeat Channel for different type of communications.
- They also store an array of peer nodes along with their details (though, just storing channels information is enough as it is implemented in such a way to handle abstraction).
- Each node will start running in an Infinite Go routine.
- The State Machine involves three states for Each Node, FOLLOWR state, CANDIDATE state and LEADER state
- Intially all the nodes starts as FOLLOWERS.
- But as the time goes on, they will be on time out and becomes CANDIDATE.
- By increasing their term, the node that became CANDIDATE will ask for Vote Request (term + its id + its own voting channel) to all its peers via their vote Request Channel.
- The nodes that recieve vote request will handle the request depending their own state and term and send the response (vote/reject) via the response vote channel of requester. The response channel is embedded in the request itself, so they dont need to look their peer information for it!
- If a CANDIDATE receives  more than 1/2 number of nodes of votes, it will become a LEADER. otherwise, it either waits for Election timout or pings from other nodes with higher terms.


## Note:
- Note-1: Replicated state is not implemented

- Note-2: Might exhibit weird behavior as routine's runtime is uncontrollable
        
- Note-3: changing timeouts should be handled crefully as algorithm most dependent on them
        
- Note-4: this is an implementation to my understanding and simplified to be a fit for Go routines and channels



        Note: any suggestions and advices are welcomed! Please make a pull request or create an issue to discuss more!
        
# Thank you!
    