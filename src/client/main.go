package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	"bkr-go/transport/message"
)

type Client struct {
	startId     int32
	port        int
	payloadSize int
	rate        int
	testTime    int
	reqNum      int
	outputFile  string
	startTime   map[int32]uint64
	replyChan   chan *message.NodeReply
	endChan     chan struct{}
	lock        sync.RWMutex
}

func (c *Client) startRpcServer() {
	rpc.Register(c)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(c.port+100))
	if err != nil {
		panic(err)
	}
	go http.Serve(listener, nil)
}

func (c *Client) NodeFinish(msg *message.NodeReply, resp *message.Response) error {
	c.replyChan <- msg
	return nil
}

func (c *Client) dealReply() {
	// file, err := os.OpenFile(c.outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0666)
	// if err != nil {
	// 	panic(err)
	// }
	// defer file.Close()
	for {
		select {
		case msg := <-c.replyChan:
			start := int32(msg.StartID)
			// for i := 0; i < int(msg.ReqNum)/c.reqNum; i++ {
			c.lock.RLock()
			fmt.Printf("recv: %d %d %d %d\n", start, c.reqNum, uint64(time.Now().UnixNano()/1000000), uint64(time.Now().UnixNano()/1000000)-c.startTime[start])
			c.lock.RUnlock()
			// 	start += int32(c.reqNum)
			// }
		case <-c.endChan:
			return
		}
	}
}

func (c *Client) run() {
	testTimer := time.NewTimer(time.Duration(c.testTime*1000) * time.Millisecond)
	fmt.Println("test start")
	payload := make([]byte, c.payloadSize*c.reqNum)
	cli, err := rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(c.port))
	if err != nil {
		panic(err)
	}
	c.lock.Lock()
	c.startTime[c.startId] = uint64(time.Now().UnixNano() / 1000000)
	c.lock.Unlock()
	for {
		select {
		case <-testTimer.C:
			c.endChan <- struct{}{}
			fmt.Println("test end")
			return
		default:
			// send reqs to server
			req := &message.ClientReq{
				StartId: c.startId,
				ReqNum:  int32(c.reqNum),
				Payload: payload,
			}

			var resp message.Response
			go cli.Call("ClientMsgProcessor.Request", req, &resp)
			c.lock.Lock()
			c.startTime[c.startId] = uint64(time.Now().UnixNano() / 1000000)
			c.lock.Unlock()
			fmt.Printf("send: %d %d %d\n", c.startId, c.reqNum, time.Now().UnixNano()/1000000)
			c.startId += int32(c.reqNum)
			// c.startTime = append(c.startTime, uint64(time.Now().UnixNano()/1000000))
			time.Sleep(time.Millisecond * 50)
		}
	}
}

func main() {
	// id := flag.Int("id", 0, "client id")
	// batchSize := flag.Int("batch", 2, "batch size")
	payloadSize := flag.Int("payload", 200, "payload size")
	// keyPath := flag.String("key", "../../../crypto", "path of ECDSA private key")
	port := flag.Int("port", 6100, "url of client")
	rate := flag.Int("rate", 60, "test time")
	testTime := flag.Int("time", 60, "test time")
	output := flag.String("output", "client.log", "output file")
	flag.Parse()

	c := &Client{
		startId:     1,
		port:        *port,
		payloadSize: *payloadSize,
		rate:        *rate,
		testTime:    *testTime,
		outputFile:  *output,
		startTime:   make(map[int32]uint64),
		replyChan:   make(chan *message.NodeReply, 1024),
		endChan:     make(chan struct{}),
		lock:        sync.RWMutex{},
	}
	c.reqNum = c.rate / 20
	c.startRpcServer()
	go c.dealReply()
	c.run()

}
