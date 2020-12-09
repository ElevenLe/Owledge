package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
)

// uniq id
// editor is a user id
// parents indicate other nodeID
// children is list of other nodeID
// related is list of other nodeID
// comments is list of other commentsID
// recommandationList is list of other nodeID
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
	ConnType string
	// content is marshal json data
	Content []byte
}

// this is the type send back
type Response struct {
	// response type is response succeess or not
	ResponseType string
	// content is marshal json data
	Content []byte
}

var wg sync.WaitGroup

var otherMachine []string

func main() {
	thisPort, backendPorts := initialBackendServer()

	otherMachine = backendPorts

	app := iris.Default()
	thisPort = ":" + thisPort

	// read the main page
	app.Get("/computerScience", func(ctx iris.Context) {
		getComputerScienceOverViewHandlerFront(ctx)
	})

	// read the node page
	app.Get("/computerScience/{nodeID}", func(ctx iris.Context) {
		getNodePageHandlerFront(ctx)
	})

	// add new comments in following node page
	app.Post("/computerScience/{nodeID}", func(ctx iris.Context) {
		createNewCommentHandlerFront(ctx)
	})

	// update the comments in this node page
	app.Post("/computerScience/{nodeID}/{commentID}/update", func(ctx iris.Context) {
		updateCommentHandlerFront(ctx)
	})

	// delete the comments
	app.Post("/computerScience/{nodeID}/delete", func(ctx iris.Context) {
		deleteCommentHandlerFront(ctx)
	})

	go circulatePing(thisPort)

	app.Listen(thisPort)
	wg.Wait()

}

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

// find the leader one time
func locateLeaderMachine(backendPorts []string, activeLeader chan string) {
	for {
		time.Sleep(1 * time.Second)
		for _, port := range backendPorts {
			wg.Add(1)
			go func(port string) {
				defer wg.Done()
				var content []byte
				res := sendData("checkLeader", content)
				if res.ResponseType == "leader" {
					activeLeader <- port
				}
				return
			}(port)
		}
	}
}

// failure detector
func circulatePing(local string) {
	pingS := make(chan struct{}, 1)
	for {
		time.Sleep(3 * time.Second)
		for _, port := range otherMachine {
			go ping(port, local, pingS)
		}

		select {
		// compete see who can arrival first.
		// if is timeout, then shows that detect a failur
		case <-time.After(time.Duration(10 * time.Second)):
			t := time.Now()
			log.Printf("Detected failure on "+local+" at ", t)
		case <-pingS:
			continue
		}
	}
}

// ping one time
func ping(service string, local string, pingS chan struct{}) {
	conn, err := net.Dial("tcp", service)
	if err != nil {
		// t := time.Now()
		// log.Printf("Detected connection failure on "+local+" with "+service+" at ", t)
		return
	}
	var res Response
	connData := Connection{
		ConnType: "checkLeader",
	}
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	encoder.Encode(connData)
	decoder.Decode(&res)
	// if ping success, then will go channel
	if res.ResponseType == "leader" {
		pingS <- struct{}{}
	}
}

func sendData(method string, content []byte) Response {
	activeLeader := make(chan string)
	activeMachine := make(chan string, len(otherMachine))
	// activeMachine := make(chan string, len(otherMachine))
	for _, port := range otherMachine {
		wg.Add(1)
		go func(port string, activeLeader chan string, activeMachine chan string) {
			defer wg.Done()
			var content []byte
			res, err := sendDataWithPort(port, "checkLeader", content)
			if err == nil {
				log.Printf("Response with ", res.ResponseType)
				if res.ResponseType == "leader" {
					activeLeader <- port
					activeMachine <- port
					return
				}
				activeMachine <- port
			}
			return
		}(port, activeLeader, activeMachine)
	}

	if method == "getNode" || method == "getAll" {
		for {
			select {
			case livePort := <-activeMachine:
				res, err := sendDataWithPort(livePort, method, content)
				if err != nil {
					res := Response{
						ResponseType: "fail",
					}
					return res
				}
				return res
			case <-time.After(time.Duration(10 * time.Second)):
				log.Printf("timeout")
				res := Response{
					ResponseType: "fail",
				}
				return res
			}
		}
	} else {
		for {
			select {
			case leaderPort := <-activeLeader:
				log.Printf("recived leader port", leaderPort)
				res, err := sendDataWithPort(leaderPort, method, content)
				if err != nil {
					res := Response{
						ResponseType: "fail",
					}
					return res
				}
				return res
			case <-time.After(time.Duration(10 * time.Second)):
				log.Printf("timeout")
				res := Response{
					ResponseType: "fail",
				}
				return res
			}
		}
	}
}

// service is where to connect
// method is connection direction
// content should be marshal json data for process service
func sendDataWithPort(service string, method string, content []byte) (Response, error) {
	// service port will be handle by the channal here
	conn, err := net.Dial("tcp", service)
	var res Response
	if err != nil {
		return res, err
	}
	defer conn.Close()
	// package the data
	connData := Connection{
		ConnType: method,
		Content:  content,
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// encoder send package to backend
	encoder.Encode(connData)
	// decoder get response from backend after data is sent
	decoder.Decode(&res)
	if service == ":8090" {
		log.Printf("8090 response data :", res.ResponseType)
	}
	return res, nil
}

// delete the target the comment
func deleteCommentHandlerFront(ctx iris.Context) {

	fmt.Println("post requset delete comment from web")

	ctx.Header("Content-Type", "application/json")
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type,Accept,X-Total,X-Limit,X-Offset")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,HEAD")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Expose-Headers", "Content-Length,Content-Encoding,Content-Type")

	var deleteContent struct {
		CommentID string "commentID"
		AuthorID  string "authorID"
	}

	err := ctx.ReadJSON(&deleteContent)
	if err != nil {
		panic(err)
		return
	}

	// package the meta data for process backend services

	// marshal the data
	marshalContent, _ := json.Marshal(deleteContent)

	res := sendData("delete", marshalContent)

	if res.ResponseType == "fail" {
		ctx.StatusCode(iris.StatusBadRequest)
	} else if res.ResponseType == "success" {
		ctx.StatusCode(iris.StatusOK)
	}

}

// update the target comment
func updateCommentHandlerFront(ctx iris.Context) {

	fmt.Println("post requset update comment from web")

	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type,Accept,X-Total,X-Limit,X-Offset")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,HEAD")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Expose-Headers", "Content-Length,Content-Encoding,Content-Type")

	// get data from web request
	commentID := ctx.Params().Get("commentID")
	var updateInformation struct {
		NewSummary string `json:"Summary"`
		NewContent string `json:"Content"`
	}
	err := ctx.ReadJSON(&updateInformation)
	if err != nil {
		panic(err)
		return
	}

	// package the meta data for process backend services
	var content struct {
		CommentID  string
		NewSummary string
		NewContent string
	}

	content.CommentID = commentID
	content.NewSummary = updateInformation.NewSummary
	content.NewContent = updateInformation.NewContent

	marshalContent, _ := json.Marshal(content)
	res := sendData("updateComment", marshalContent)

	if res.ResponseType == "fail" {
		ctx.StatusCode(iris.StatusBadRequest)
	} else if res.ResponseType == "success" {
		ctx.StatusCode(iris.StatusOK)
	}
}

// create a new comment
func createNewCommentHandlerFront(ctx iris.Context) {

	fmt.Println("post requset create comment from web")

	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type,Accept,X-Total,X-Limit,X-Offset")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,HEAD")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Expose-Headers", "Content-Length,Content-Encoding,Content-Type")

	if ctx.Method() == "OPTIONS" {
		return
	}

	var newComment Comment
	err := ctx.ReadJSON(&newComment)
	if err != nil {
		panic(err)
		return
	}

	// package the meta data for process backend services
	type content struct {
		NodeID     string
		Newcomment Comment
	}

	sendContent := &content{
		NodeID:     ctx.Params().Get("nodeID"),
		Newcomment: newComment,
	}
	marshalContent, err := json.Marshal(sendContent)
	res := sendData("createComment", marshalContent)

	if res.ResponseType == "fail" {
		ctx.StatusCode(iris.StatusBadRequest)
	} else if res.ResponseType == "success" {
		ctx.StatusCode(iris.StatusOK)
	}
}

// get general pages
func getComputerScienceOverViewHandlerFront(ctx iris.Context) {

	fmt.Println("get requset general info from web")

	ctx.Header("Content-Type", "application/json")
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type,Accept,X-Total,X-Limit,X-Offset")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,HEAD")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Expose-Headers", "Content-Length,Content-Encoding,Content-Type")

	if ctx.Method() == "OPTIONS" {
		return
	}

	// package the meta data for process backend services
	empty := make([]byte, 0)
	res := sendData("getAll", empty)

	if res.ResponseType == "fail" {
		ctx.StatusCode(iris.StatusBadRequest)
	} else if res.ResponseType == "success" {
		var generalPost struct {
			Nodes []Node `json:"nodes"`
		}
		json.Unmarshal(res.Content, &generalPost)
		ctx.JSON(generalPost)
	}
}

// get one onde page info
func getNodePageHandlerFront(ctx iris.Context) {

	fmt.Println("get requset one node info from web")

	ctx.Header("Content-Type", "application/json")
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type,Accept,X-Total,X-Limit,X-Offset")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,HEAD")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Expose-Headers", "Content-Length,Content-Encoding,Content-Type")

	// package the meta data for process backend services
	nodeID := ctx.Params().Get("nodeID")

	content, _ := json.Marshal(nodeID)

	res := sendData("getNode", content)

	if res.ResponseType == "fail" {
		ctx.StatusCode(iris.StatusBadRequest)
	} else if res.ResponseType == "success" {
		var nodePost struct {
			Node     Node
			Editor   User
			Comments []Comment
		}
		json.Unmarshal(res.Content, &nodePost)
		ctx.JSON(nodePost)
	}
}
