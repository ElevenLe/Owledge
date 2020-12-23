package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const HEARTBEAT = 10

// this backend server support reft as consensus model for distributed backend

// -------------------------------------------------------- //
// 						Reft implementation 				//
// machine state keep track
type Machine struct {
	leader               bool
	candidate            bool
	follower             bool
	rigthOfVote          bool
	term                 int
	electionVote         int
	selectLeader         string
	port                 string
	otherMachineLocation []string
	operationLog         []Operation // machine log for keep track operation from frontend server
	operationPosition    int
}

type Operation struct {
	Log  string
	Data []byte
}

// following is the messages format that sends out through TCP
type RequestVote struct {
	ConnSource string
	Term       int
}

// vote is sendout type for voting process
type Vote struct {
	ConnSource string
	Voting     bool
}

type AppendEntry struct {
	ConnSource   string
	Term         int
	OperationLog []Operation
}

//
type AckAppend struct {
	ConnSource string
	AckType    bool
}

type CommitLog struct {
	ConnSource string
	Term       int
	Position   int
}

var role chan int

func getElectionTimeOut() int {
	rand.Seed(time.Now().UnixNano())
	min := 150
	max := 300
	time := rand.Intn(max-min) + min
	return time
}

// reft functionality
// initial a machine
func initialMachine() *Machine {
	newMachine := &Machine{
		leader:       false,
		candidate:    false,
		follower:     true,
		term:         0,
		electionVote: 0,
		selectLeader: "none",
	}

	initalOperation := Operation{
		Log: "initial",
	}

	newMachine.operationLog = append(newMachine.operationLog, initalOperation)
	newMachine.operationPosition = 0

	log.Printf("Raft: initial raft in this machine")
	return newMachine
}

// this is raft started function.
// this function helps to change the roles
func startRaft(this *Machine) {
	role = make(chan int, 1)
	if this.follower {
		role <- 1
	}
	log.Printf("Raft: started raft")
	for {
		select {
		case state := <-role:
			wg.Add(1)
			if state == 1 {
				log.Printf("Raft: role change to followers")
				go followerProcess(this, role)
			} else if state == 2 {
				log.Printf("Raft: role change to candidate")
				go candidateProcess(this, role)
			} else {
				log.Printf("Raft: role change to leader")
				go leaderProcess(this, role)
			}
		case readRequest := <-frontendChannelRead:
			log.Printf("------DEBUG 1:")
			go func(readRequest Connection) {
				addOperationLog(this, readRequest)
				response := logCommit(this, len(this.operationLog)-1)
				// tell frontend i have commited, and give data to it
				responseChannel <- response
				// if recieve commit requset from master, then do the log commit
				return
			}(readRequest)
		}
	}
}

// keep track heartbeat timeout.
// heartbeat timeout is how long for waiting for a heartbeat from a leader
// if heartbeat timeouts, then start
// when follower roles begin
func followerProcess(this *Machine, role chan int) {
	log.Printf("Raft: Follower Process started")
	this.leader = false
	this.candidate = false
	this.follower = true
	for {
		select {
		// if heartBeat timeout: change role to candidate
		// !!Hearbeat Timeout!!
		case <-time.After(time.Duration(HEARTBEAT * time.Second)):
			log.Printf("Raft--Follower Process: heartbeat timeout")
			role <- 2
			return
		// if inside heartBeat timeout, recieved AppendEntry: heartBeat timeout reset
		case leaderAppendEntry := <-appendEntryChannel:
			log.Printf("Raft--Follower Process: recieve heartBeat")
			if leaderAppendEntry.Term >= this.term {
				this.operationLog = leaderAppendEntry.OperationLog
				this.term = leaderAppendEntry.Term
				// ackAppendEntry(this, leaderAppendEntry)
				// Send ack from RPC center
			}
		case leaderCommit := <-commitChannel:
			log.Printf("Raft--Follower Process: commit")
			if leaderCommit.Term >= this.term {
				logCommit(this, leaderCommit.Position)
			}
		}
	}
}

// keep track election timeout.
// election timeout is how long a follower could become a candidate
// if election timeouts, then this follower become candidate
// and start ask for vote
// if vote, then restart this timeout
// when candident roles begin
func candidateProcess(this *Machine, role chan int) {
	log.Printf("Raft--Candidate Process started")
	this.leader = false
	this.candidate = true
	this.follower = false
	for {
		electionTimeout := getElectionTimeOut()
		select {
		// after election timeout, the node become candidate
		// 1. start new election term
		// 2. send request vote to others
		// 3. vote for it self
		// !!Election Timeout!!
		case <-time.After(time.Duration(electionTimeout) * time.Millisecond):
			log.Printf("Raft--Candidate Process: election timeout")
			this.term = this.term + 1
			otherVotes := requestVote(this)
			this.electionVote = this.electionVote + 1 + otherVotes
			if this.electionVote > len(this.otherMachineLocation)/2 {
				this.electionVote = 0
				log.Printf("Raft--Candidate Process: this machine selected to Leader!")
				role <- 3
			} else {
				this.electionVote = 0
				log.Printf("Raft--Candidate Process: redo election timeout")
				role <- 2
			}
			return
		// if within timeout, this machine recieved a request vote
		// then it should send vote to the first machine that request vote
		case request := <-requsetVoteChannel:
			// see if the request term is larger than this term
			log.Printf("Raft--Candidate Process: recieved a vote request")
			if request.Term > this.term {
				// vote for other machine from RPC center
				// add current term
				this.term = this.term + 1
				role <- 1
				return
			}
		case leader := <-appendEntryChannel:
			log.Printf("Raft--Candidate Process: recieved a heartBeat")
			if leader.Term >= this.term {
				this.term = leader.Term
				this.operationLog = leader.OperationLog
				role <- 1
				return
			}
		}
	}
}

// when leader roles begin
func leaderProcess(this *Machine, role chan int) {
	log.Printf("Raft: Leader Process started")
	this.leader = true
	this.candidate = false
	this.follower = false
	for {
		// this machine already recieved the major vote
		// need to send out the AppendRequest in time interval
		select {
		// if leader recieves message from frontend
		case frontendRequest := <-frontendChannelWrite:
			log.Printf("Raft--Leader: recieve fronted request")
			switch frontendRequest.ConnType {
			case "checkLeader":
				checkResponse := Response{
					ResponseType: "leader",
				}
				responseChannel <- checkResponse
			default:
				// add operation to self log
				addOperationLog(this, frontendRequest)
				// tell followers to copy the log
				acksAppend := appendEntry(this)
				// if recieve acks from all machine, commit
				if acksAppend+1 > len(this.otherMachineLocation)/2 {
					// tell myself to commit
					response := logCommit(this, len(this.operationLog)-1)
					// tell frontend i have commited, and give data to it
					responseChannel <- response
					// tell other followers to commit
					commitLog(this, this.operationPosition)
				} else {
					res := Response{
						ResponseType: "fail",
					}
					responseChannel <- res
				}
			}
		// if leader recieves other leader append
		// if other leader term is larger
		// change this leader to follower
		case leader := <-appendEntryChannel:
			log.Printf("Raft--Leader: recieve HeartBeat")
			if leader.Term > this.term {
				this.term = leader.Term
				this.operationLog = leader.OperationLog
				role <- 1
				return
			}
		default:
			// send heartbeat in time interval
			log.Printf("Raft--Leader: send HeartBeat")
			appendEntry(this)
			time.Sleep(1 * time.Second)
		}
	}
}

// when election term started, and current node become candidate,
// start reqeust vote, and vote for themself.
// return how many votes that comes from other machine
func requestVote(this *Machine) int {
	// inital a vote channel for collecting vote
	voteChan := make(chan int, len(this.otherMachineLocation))
	log.Printf("Raft--Election--RequestVote: started")
	// loop over the others machine port
	for _, port := range this.otherMachineLocation {
		go func(port string) {
			log.Printf("Raft--Election--RequestVote: send to", port)
			conn, err := net.Dial("tcp", port)
			if err != nil {
				return
			}
			defer conn.Close()
			// send request vote notification
			data := RequestVote{
				ConnSource: this.port,
				Term:       this.term,
			}
			marshalData, _ := json.Marshal(data)
			connRequest := Connection{
				ConnType: "requestVote",
				Content:  marshalData,
			}
			encoder := json.NewEncoder(conn)
			encoder.Encode(connRequest)

			// hanle votes from followers
			var vote Vote
			decoder := json.NewDecoder(conn)
			decoder.Decode(&vote)
			if vote.Voting {
				log.Printf("Raft--Election--RequestVote: recieve from", port)
				voteChan <- 1
			}
			return
		}(port)
	}

	// determing in election timeout, recieves how many votes
	vote := 0
	for {
		select {
		case <-voteChan:
			vote = vote + 1
			if vote > len(this.otherMachineLocation)/2 {
				close(voteChan)
				return vote
			}
		case <-time.After(time.Duration(1 * time.Second)):
			close(voteChan)
			return vote
		}
	}
}

// if is follower and recieves the requestVote,
// 1. start vote the first machine that send the requset
// 2. add term
// 3. close vote right
func vote(this *Machine, conn net.Conn) {
	log.Printf("Raft--Election--Voting: started")
	data := Vote{
		ConnSource: this.port,
		Voting:     true,
	}
	// marshalData, _ := json.Marshal(data)
	// connRequest := Connection{
	// 	ConnType: "vote",
	// 	Content:  marshalData,
	// }
	encoder := json.NewEncoder(conn)
	encoder.Encode(data)
}

// add front end operation to the log
func addOperationLog(this *Machine, frontendRequest Connection) {
	newOperation := Operation{
		Log:  frontendRequest.ConnType,
		Data: frontendRequest.Content,
	}
	this.operationLog = append(this.operationLog, newOperation)
}

// after current machine become the leader
// started to send appendEntry in time interval
// response to celect the acks from all other machine
// if one machine did not send ack, then do not commit
// if there is a timeout waiting for some machine, do not commit
func appendEntry(this *Machine) int {
	log.Printf("Raft--AppendEntry: started")
	ackChan := make(chan int, len(this.otherMachineLocation))
	for _, port := range this.otherMachineLocation {
		go func(ackChan chan int, port string) {
			conn, err := net.Dial("tcp", port)
			if err != nil {
				return
			}
			log.Printf("Raft--AppendEntry: send to", port)
			defer conn.Close()
			// send append entry as heartbeat
			// should i send log data?
			data := AppendEntry{
				ConnSource:   this.port,
				Term:         this.term,
				OperationLog: this.operationLog,
			}
			// can marshalled data been marshal again?
			marshalData, _ := json.Marshal(data)
			var UnmarshalData AppendEntry
			json.Unmarshal(marshalData, &UnmarshalData)
			log.Printf("-----------------DEBUG BREAK 5 -- PASS")
			connRequest := Connection{
				ConnType: "appendEntry",
				Content:  marshalData,
			}
			encoder := json.NewEncoder(conn)
			encoder.Encode(connRequest)

			// recieve ack from followers
			var ack AckAppend
			decoder := json.NewDecoder(conn)
			decoder.Decode(&ack)
			if ack.AckType {
				log.Printf("Raft--AppendEntry: recieved from", port)
				ackChan <- 1
			}
		}(ackChan, port)
	}

	ackNum := 0
	for {
		select {
		case <-ackChan:
			ackNum = ackNum + 1
			if ackNum == len(this.otherMachineLocation) {
				close(ackChan)
				return ackNum
			}
			// heartbeat timeout
		case <-time.After(time.Duration(3 * time.Second)):
			return ackNum
		}
	}
}

// if recieve the appendEntry request
// current machine, as follower, should change the log
// and send ack back to leader
func ackAppendEntry(this *Machine, conn net.Conn) {
	log.Printf("Raft--Acking: started")
	data := AckAppend{
		ConnSource: this.port,
		AckType:    true,
	}
	// marshalData, _ := json.Marshal(data)
	// connRequest := Connection{
	// 	ConnType: "ackAppend",
	// 	Content:  marshalData,
	// }
	encoder := json.NewEncoder(conn)
	encoder.Encode(data)
}

// current machine is leader
// after sends appendEntry reqeust, recieve majority of ack
// send commit to all follower
func commitLog(this *Machine, position int) {
	log.Printf("Raft--Commit: started")
	for _, port := range this.otherMachineLocation {
		go func(port string) {
			conn, err := net.Dial("tcp", port)
			if err != nil {
				return
			}
			log.Printf("Raft--Commit: send to", port)
			defer conn.Close()
			// send commit request to all
			data := CommitLog{
				ConnSource: this.port,
				Term:       this.term,
				Position:   position,
			}
			marshalData, _ := json.Marshal(data)
			connRequest := Connection{
				ConnType: "commit",
				Content:  marshalData,
			}
			encoder := json.NewEncoder(conn)
			encoder.Encode(connRequest)
		}(port)
	}
}

// this machine do the log opertaion process
// if leader recieves a reqeust from frontend
// if follower recieves a commitLog from leader
// both condition will do the logCommit
func logCommit(this *Machine, lastPosition int) Response {
	// is current machine is leader, then send response
	// if current machine is not leader, then return nil
	var response Response
	// this operation position is where the last operation has done in this machine
	// lastPosistion is where leader machine last operation has done

	// log.Printf("log Commit on: ", this.port)
	// log.Printf("this log position: ", this.operationPosition)
	// log.Printf("asked for position: ", lastPosition)
	for lastPosition > this.operationPosition {
		this.operationPosition++
		response = logCommitOnce(this, this.operationPosition)
	}
	return response
}

func logCommitOnce(this *Machine, commitPosition int) Response {
	oneOperation := this.operationLog[commitPosition]
	return processFrontendServices(oneOperation.Log, oneOperation.Data)
}

// after success commit in the follower,
// sends back the response.
func responseClient(this *Machine, conn net.Conn, res Response) {
	log.Printf("Raft--ResponseFront: started")
	encoder := json.NewEncoder(conn)
	encoder.Encode(res)
}

// -------------------------------------------------------- //
// 					Backend servers 						//
type Node struct {
	Id                 string `json:"nodeID"`
	Title              string
	Editor             string
	Content            string
	Parents            string
	Children           []string
	Related            []string
	Comments           []string
	RecommandationList []string
}

type Comment struct {
	Id       string `json:"commentID"`
	NodeID   string
	AuthorID string
	Summary  string
	Content  string
	Likes    string
}

// comments is list of other commentsID
type User struct {
	Id       string `json:"userID"`
	Name     string
	Comments []string
}

// connection is type received
type Connection struct {
	// conntype determine which services
	// frontend list: delete, updateComment, createComment, getAll, getNode
	// Raft list: requestVote, appendEntry, commit
	ConnType string
	// content is marshal json data
	// this content could be used by both frontend or other backend machines
	// if is other backend machine, then the request of RAFT is trans to json data
	// if is frontend, then the reqeust of servers info as json data
	Content []byte
}

// this is the type send back
type Response struct {
	// response type is response succeess or not
	ResponseType string
	// content is marshal json data
	Content []byte
}

var globalNode []*Node
var globalUser []*User
var globalComment []*Comment
var machine *Machine

var tableLock = sync.Mutex{}
var wg sync.WaitGroup

var requsetVoteChannel chan RequestVote
var appendEntryChannel chan AppendEntry
var commitChannel chan CommitLog
var frontendChannelRead chan Connection
var frontendChannelWrite chan Connection
var responseChannel chan Response

// program entry
func main() {
	// inital raft state
	machine = initialMachine()

	// inital all global data
	globalNode, globalUser, globalComment = initalGloableData()

	// read from commend about port information
	thisPort, othersPort := initialBackendServer()

	// listen at current machine
	machine.port = thisPort
	machine.otherMachineLocation = othersPort

	// inital all the raft channel
	requsetVoteChannel = make(chan RequestVote, 1)
	appendEntryChannel = make(chan AppendEntry, 1)
	commitChannel = make(chan CommitLog, 1)
	frontendChannelRead = make(chan Connection, 1)
	frontendChannelWrite = make(chan Connection, 1)
	responseChannel = make(chan Response, 1)

	// start Raft process
	wg.Add(1)
	go startRaft(machine)

	// start PRC accepting process
	// this process is response for keep recieving request from other machine, no matter front end or other backend
	wg.Add(1)
	go acceptingRPC(machine)

	wg.Wait()
	fmt.Println(machine)
}

// detecting the args
// backend --listen 8090 --backend :8091 ,:8092
func initialBackendServer() (string, []string) {
	args := os.Args[1:]
	currentMachinePort := ""
	var otherMachineList []string
	if len(args) == 4 {
		listenFlag := args[0]
		if listenFlag == "--listen" {
			currentMachinePort = args[1]
			log.Printf("current machine listen: " + currentMachinePort)
		} else {
			panic("Commendline: need --listen peremeter. Please try again")
		}
		othersFlag := args[2]
		if othersFlag == "--backend" {
			otherMachine := args[3]
			otherMachineList = strings.Split(otherMachine, ",")
			log.Printf("others machine location read")
			fmt.Println(otherMachineList)
		} else {
			panic("Commendline: need --backend peremeter. Please try again")
		}
	} else {
		panic("Commendline: please try again with input --listen and --backend")
	}

	return currentMachinePort, otherMachineList
}

func acceptingRPC(this *Machine) {
	backendServer, err := net.Listen("tcp", ":"+this.port)
	if err != nil {
		panic("Server: server not established")
	}
	for {
		conn, err := backendServer.Accept()
		if err != nil {
			panic("Server: connection error")
		}
		wg.Add(1)
		go func(this *Machine, conn net.Conn) {
			defer wg.Done()
			defer conn.Close()
			var connRequest Connection
			decoder := json.NewDecoder(conn)
			// determing the request here
			decoder.Decode(&connRequest)
			switch connRequest.ConnType {
			case "requestVote":
				var requestVote RequestVote
				json.Unmarshal(connRequest.Content, &requestVote)
				if requestVote.Term > this.term {
					log.Printf("RPC center: recieved RequestVote from", requestVote.ConnSource)
					requsetVoteChannel <- requestVote
					vote(this, conn)
				}
			case "appendEntry":
				var appendEntry AppendEntry
				json.Unmarshal(connRequest.Content, &appendEntry)
				log.Printf("DEBUG : ", appendEntry.OperationLog)
				log.Printf("RPC center: recieved AppendEntry from", appendEntry.ConnSource)
				if appendEntry.Term >= this.term {
					appendEntryChannel <- appendEntry
					ackAppendEntry(this, conn)
				}
			case "commit":
				var commitData CommitLog
				json.Unmarshal(connRequest.Content, &commitData)
				log.Printf("RPC center: recieved Commit from", commitData.ConnSource)
				commitChannel <- commitData
			case "getNode":
				log.Printf("RPC center: recieved FrontEnd: getNode")
				frontendChannelRead <- connRequest
				response := <-responseChannel
				responseClient(this, conn, response)
			case "getAll":
				log.Printf("RPC center: recieved FrontEnd: getAll")
				frontendChannelRead <- connRequest
				response := <-responseChannel
				responseClient(this, conn, response)
			default:
				log.Printf("RPC center: recieved FrontEnd")
				if this.leader {
					frontendChannelWrite <- connRequest
					select {
					case response := <-responseChannel:
						responseClient(this, conn, response)
					case <-time.After(time.Duration(10 * time.Second)):
						response := Response{
							ResponseType: "fail",
						}
						responseClient(this, conn, response)
					}
				} else {
					response := Response{
						ResponseType: "InactiveLeader",
					}
					responseClient(this, conn, response)
				}
			}
			return
		}(this, conn)
	}
}

// gloabl variables
// connContent used for determine which service to use
// content in the Connection provides info that services required
func processFrontendServices(connType string, connData []byte) Response {
	res := make(chan Response)
	// determine which service to use
	switch connType {
	case "delete":
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := deleteCommentHandlerBack(connData)
			res <- data
		}()
	case "updateComment":
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := updateCommentHandlerBack(connData)
			res <- data
		}()
	case "createComment":
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := createNewCommentHandlerBack(connData)
			res <- data
		}()
	case "getAll":
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := getComputerScienceOverViewHandlerBack(connData)
			res <- data
		}()
	case "getNode":
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := getNodePageHandlerBack(connData)
			res <- data
		}()
	default: // if front end server send random command, process as a fail
		data := Response{
			ResponseType: "fail",
		}
		res <- data
	}
	return <-res
}

// content here contains a marshaled json data
func deleteCommentHandlerBack(content []byte) Response {
	// fmt.Println("delete")
	// content is a marshal type, but need to convert to json
	var deleteContent struct {
		CommentID string
		AuthorID  string
	}
	json.Unmarshal(content, &deleteContent)

	// fmt.Println(deleteContent)
	// nodeID := deleteContent.NodeID
	commentID := deleteContent.CommentID
	authorID := deleteContent.AuthorID

	successLog := false
	tableLock.Lock()
	for index, comment := range globalComment {
		if comment != nil {
			if comment.Id == commentID && comment.AuthorID == authorID {
				copy(globalComment[index:], globalComment[index+1:])
				globalComment[len(globalComment)-1] = nil
				globalComment = globalComment[:len(globalComment)-1]
				successLog = true
			}
		}
	}
	tableLock.Unlock()

	var res Response
	if successLog {
		res = Response{
			ResponseType: "success",
		}
	} else {
		res = Response{
			ResponseType: "fail",
		}
	}

	return res
}

// content contains a marshal json data
func updateCommentHandlerBack(content []byte) Response {
	// fmt.Println("udpate")
	// content is a marshal type, but need to convert to json
	var updateInformation struct {
		CommentID  string
		NewSummary string
		NewContent string
	}

	json.Unmarshal(content, &updateInformation)

	successLog := false
	tableLock.Lock()
	for _, comment := range globalComment {
		if comment != nil {
			if comment.Id == updateInformation.CommentID {
				if updateInformation.NewSummary != "" {
					comment.Summary = updateInformation.NewSummary
				}
				if updateInformation.NewContent != "" {
					comment.Content = updateInformation.NewContent
				}
				successLog = true
			}
		}
	}
	tableLock.Unlock()

	var res Response
	if successLog {
		res = Response{
			ResponseType: "success",
		}
	} else {
		res = Response{
			ResponseType: "fail",
		}
	}

	return res
}

// content contains a marshal json data
func createNewCommentHandlerBack(content []byte) Response {
	// fmt.Println("create")
	// content is a marshal type, but need to convert to json
	var frontComment struct {
		NodeID     string
		Author     string
		Newcomment Comment
	}

	json.Unmarshal(content, &frontComment)

	nodeID := frontComment.NodeID
	newComment := frontComment.Newcomment
	// authorID := frontComment.Author

	// add to user's comment
	// var currUser *User

	successLog := false
	tableLock.Lock()
	for _, node := range globalNode {
		if node.Id == nodeID {
			node.Comments = append(node.Comments, newComment.Id)
			successLog = true
		}
	}

	// add to global comments list
	pointer := &frontComment.Newcomment
	globalComment = append(globalComment, pointer)
	tableLock.Unlock()

	var res Response
	if successLog {
		res = Response{
			ResponseType: "success",
		}
	} else {
		res = Response{
			ResponseType: "fail",
		}
	}

	return res
}

// content contains a marshal json data
func getComputerScienceOverViewHandlerBack(content []byte) Response {
	// fmt.Println("Get All")
	// content is a marshal type, but need to convert to json
	var generalPost struct {
		Nodes []Node
	}

	var nodes []Node

	successLog := false
	tableLock.Lock()
	for _, node := range globalNode {
		nodes = append(nodes, *node)
	}
	successLog = true
	tableLock.Unlock()

	generalPost.Nodes = nodes

	marshalContent, _ := json.Marshal(generalPost)

	var res Response
	if successLog {
		res = Response{
			ResponseType: "success",
			Content:      marshalContent,
		}
	} else {
		res = Response{
			ResponseType: "fail",
		}
	}
	return res
}

// content contains a marshal json data
func getNodePageHandlerBack(content []byte) Response {
	// fmt.Println("Get Node")
	// content here is a nodeID as string type
	var nodeID string

	json.Unmarshal(content, &nodeID)

	var node *Node
	var editor *User
	var comments []Comment

	successLog := false
	tableLock.Lock()
	for _, elem := range globalNode {
		if elem.Id == nodeID {
			node = elem
			successLog = true
		}
	}
	tableLock.Unlock()
	tableLock.Lock()
	if node != nil {
		for _, elem := range globalUser {
			if node.Editor == elem.Id {
				editor = elem
				successLog = true
			}
		}
	}
	tableLock.Unlock()
	tableLock.Lock()
	if node != nil {
		for _, elem := range globalComment {
			if elem != nil {
				if node.Id == elem.NodeID {
					comments = append(comments, *elem)
					successLog = true
				}
			}
		}
	}
	tableLock.Unlock()

	var nodePost struct {
		Node     Node
		Editor   User
		Comments []Comment
	}

	if node != nil {
		nodePost.Node = *node
	}

	if editor != nil {
		nodePost.Editor = *editor
	}

	if len(comments) > 0 {
		nodePost.Comments = comments
	}

	marshalPost, _ := json.Marshal(nodePost)

	var res Response
	if successLog {
		res = Response{
			ResponseType: "success",
			Content:      marshalPost,
		}
	} else {
		res = Response{
			ResponseType: "fail",
		}
	}

	return res

}

// inital all gloable data
func initalGloableData() ([]*Node, []*User, []*Comment) {
	node1 := &Node{
		Id:      "1",
		Title:   "Computer Science",
		Content: "Computer science is the study of computation and information. Computer science deals with theory of computation, algorithms, computational problems and the design of computer systems hardware, software and applications. Computer science addresses both human-made and natural information processes, such as communication, control, perception, learning and intelligence especially in human-made computing systems and machines.",
	}

	node2 := &Node{
		Id:      "2",
		Title:   "Distributed System",
		Content: "A distributed system is a system whose components are located on different networked computers, which communicate and coordinate their actions by passing messages to one another. The components interact with one another in order to achieve a common goal.",
	}

	node3 := &Node{
		Id:      "3",
		Title:   "CAP theorem",
		Content: "C is consistent: Every read receives the most recent write or an error, A is availability : Every request receives a (non-error) response, without the guarantee that it contains the most recent write, P is Partition tolerance: The system continues to operate despite an arbitrary number of messages being dropped (or delayed) by the network between nodes",
	}

	node4 := &Node{
		Id:      "4",
		Title:   "Algorithm",
		Content: "an algorithm is a finite sequence of well-defined, computer-implementable instructions, typically to solve a class of problems or to perform a computation. Algorithms are always unambiguous and are used as specifications for performing calculations, data processing, automated reasoning, and other tasks.",
	}

	node3.Parents = node2.Id
	node2.Parents = node1.Id
	node4.Parents = node1.Id

	node1.Children = []string{node2.Id, node4.Id}
	node2.Children = []string{node3.Id}

	globalNode := []*Node{node1, node2, node3, node4}

	user1 := &User{
		Id:   "user1",
		Name: "Zhengyi Li",
	}

	user2 := &User{
		Id:   "user2",
		Name: "Kay",
	}

	user3 := &User{
		Id: "user3",
	}

	user4 := &User{
		Id: "user4",
	}

	auth := &User{
		Id: "auth",
	}

	node1.Editor = auth.Id
	node2.Editor = auth.Id
	node3.Editor = auth.Id
	node4.Editor = auth.Id

	globalUser := []*User{user1, user2, auth, user3, user4}

	comment1 := &Comment{
		Id:       "c1",
		NodeID:   "3",
		AuthorID: "user1",
		Content:  "The CAP theorem implies that in the presence of a network partition, one has to choose between consistency and availability",
		Likes:    "10",
	}

	comment2 := &Comment{
		Id:       "c2",
		NodeID:   "2",
		AuthorID: "user2",
		Content:  "Distributed System enhence the overall performance by scaling out",
	}

	comment1.Summary = comment1.Content
	comment2.Summary = comment2.Content

	node2.Comments = []string{comment2.Id}
	node3.Comments = []string{comment1.Id}

	globalComment := []*Comment{comment1, comment2}

	return globalNode, globalUser, globalComment
}
