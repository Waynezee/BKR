package http

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"bkr-go/transport/info"
	"bkr-go/transport/message"

	"github.com/perlin-network/noise"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var (
	clientPrefix  = "/client"
	streamBufSize = 40960 * 16
)

type Peer struct {
	PeerID    uint32
	Addr      string
	PublicKey *ecdsa.PrivateKey
}

// HTTPTransport is responsible for message exchange among nodes
type HTTPTransport struct {
	id         info.IDType
	node       *noise.Node
	peers      map[uint32]*Peer
	msgc       chan *message.ConsMessage
	batchsize  int
	PrivateKey ecdsa.PrivateKey
	mu         sync.Mutex
}

type NoiseMessage struct {
	Msg *message.ConsMessage
}

func (m NoiseMessage) Marshal() []byte {
	data, err := proto.Marshal(m.Msg)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func UnmarshalNoiseMessage(buf []byte) (NoiseMessage, error) {
	m := NoiseMessage{Msg: new(message.ConsMessage)}
	err := proto.Unmarshal(buf, m.Msg)
	if err != nil {
		return NoiseMessage{}, err
	}
	return m, nil
}

// Broadcast msg to all peers
func (tp *HTTPTransport) Broadcast(msg *message.ConsMessage) {
	msg.From = uint32(tp.id)
	// Sign(msg, &tp.PrivateKey)
	tp.msgc <- msg

	for _, p := range tp.peers {
		if p != nil {
			go tp.SendMessage(p.PeerID, msg)
		}
	}
}

// InitTransport executes transport layer initiliazation, which returns transport, a channel
// for received ConsMessage, a channel for received requests, and a channel for reply
func InitTransport(lg *zap.Logger, id info.IDType, port int, peers []Peer, ck *ecdsa.PrivateKey, batchsize int) (*HTTPTransport,
	chan *message.ConsMessage, chan []byte, chan []byte, chan struct{}, chan []byte) {
	msgc := make(chan *message.ConsMessage, streamBufSize)
	tp := &HTTPTransport{id: id, peers: make(map[uint32]*Peer), msgc: msgc, batchsize: batchsize, mu: sync.Mutex{}}
	for i, p := range peers {
		if index := uint32(i); index != uint32(id) {
			tp.peers[index] = &Peer{PeerID: uint32(index), Addr: p.Addr[7:], PublicKey: p.PublicKey}
		} else {
			tp.PrivateKey = *p.PublicKey
			ip := strings.Split(p.Addr, ":")
			node_port, _ := strconv.ParseUint(ip[2], 10, 64)
			node, _ := noise.NewNode(noise.WithNodeBindHost(net.ParseIP("127.0.0.1")),
				noise.WithNodeBindPort(uint16(node_port)), noise.WithNodeMaxRecvMessageSize(32*1024*1024))
			tp.node = node
		}
	}
	tp.node.RegisterMessage(NoiseMessage{}, UnmarshalNoiseMessage)
	tp.node.Handle(tp.Handler)
	err := tp.node.Listen()
	if err != nil {
		panic(err)
	}
	log.Printf("listening on %v\n", tp.node.Addr())

	reqChan := make(chan *message.ClientReq, streamBufSize)
	proposeChan := make(chan struct{}, 100)
	reqc := make(chan []byte, streamBufSize)
	repc := make(chan []byte, streamBufSize)

	restoreChan := make(chan []byte, streamBufSize)

	rprocessor := &ClientMsgProcessor{
		lg:          lg,
		id:          id,
		tryPropose:  proposeChan,
		reqChan:     reqChan,
		restoreChan: restoreChan,
		reqc:        reqc,
		repc:        repc,
		port:        port,
		reqs:        new(message.Batch),
		batch:       batchsize,
		reqNum:      0,
		startId:     1}
	go rprocessor.run()
	// mux := http.NewServeMux()
	// mux.HandleFunc("/", http.NotFound)
	// mux.Handle(clientPrefix, rprocessor)
	// mux.Handle(clientPrefix+"/", rprocessor)
	// server := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: mux}
	// server.SetKeepAlivesEnabled(true)

	// go server.ListenAndServe()

	return tp, msgc, reqc, repc, proposeChan, restoreChan
}

func (tp *HTTPTransport) SendMessage(id uint32, msg *message.ConsMessage) {
	m := NoiseMessage{Msg: msg}
	err := tp.node.SendMessage(context.TODO(), tp.peers[id].Addr, m)
	for {
		if err == nil {
			return
		} else {
			time.Sleep(1 * time.Second)
			fmt.Println("err", err.Error())
		}
		err = tp.node.SendMessage(context.TODO(), tp.peers[id].Addr, m)
	}
}

func (tp *HTTPTransport) Handler(ctx noise.HandlerContext) error {
	obj, err := ctx.DecodeMessage()
	if err != nil {
		log.Fatal(err)
	}
	msg, ok := obj.(NoiseMessage)
	if !ok {
		log.Fatal(err)
	}
	go tp.OnReceiveMessage(msg.Msg)
	return nil
}

func (tp *HTTPTransport) OnReceiveMessage(msg *message.ConsMessage) {
	// if msg.From == uint32(tp.id) {
	// 	tp.msgc <- msg
	// 	return
	// }
	// if Verify(msg, tp.peers[msg.From].PublicKey) {
	// 	tp.msgc <- msg
	// }
	tp.msgc <- msg
}

// func Verify(msg *message.ConsMessage, pub *ecdsa.PrivateKey) bool {
// 	toVerify := &message.ConsMessage{
// 		Type:     msg.Type,
// 		From:     msg.From,
// 		Proposer: msg.Proposer,
// 		Round:    msg.Round,
// 		Sequence: msg.Sequence,
// 		Content:  msg.Content,
// 	}
// 	content, err := proto.Marshal(toVerify)
// 	if err != nil {
// 		panic(err)
// 	}

// 	hash, err := sha256.ComputeHash(content)
// 	if err != nil {
// 		panic("sha256 computeHash failed")
// 	}
// 	b, err := myecdsa.VerifyECDSA(&pub.PublicKey, msg.Signature, hash)
// 	if err != nil {
// 		fmt.Println("Failed to verify a consmsgpb: ", err)
// 	}
// 	return b
// }

// func Sign(msg *message.ConsMessage, priv *ecdsa.PrivateKey) {
// 	content, err := proto.Marshal(msg)
// 	if err != nil {
// 		panic(err)
// 	}
// 	hash, err := sha256.ComputeHash(content)
// 	if err != nil {
// 		panic("sha256 computeHash failed")
// 	}
// 	sig, err := myecdsa.SignECDSA(priv, hash)
// 	if err != nil {
// 		panic("myecdsa signECDSA failed")
// 	}
// 	msg.Signature = sig
// }

// ClientMsgProcessor is responsible for listening and processing requests from clients
type ClientMsgProcessor struct {
	num      int
	port     int
	lg       *zap.Logger
	id       info.IDType
	batch    int
	reqs     *message.Batch
	interval int
	reqNum   int
	startId  int32

	tryPropose  chan struct{}
	reqChan     chan *message.ClientReq
	reqc        chan []byte // send to proposer;
	repc        chan []byte // receive from state;
	restoreChan chan []byte // rebatch the requests which are decided 0;
}

func (cp *ClientMsgProcessor) run() {

	timer := time.NewTimer(time.Millisecond * time.Duration(100))
	cp.startRpcServer()
	go cp.ReplyServer()
	req := <-cp.reqChan
	cp.reqs.Reqs = append(cp.reqs.Reqs, req)
	cp.reqNum += cp.interval

	// makebatch := false
	for {
		select {
		// case <- timer.C:
		// 	if !proposeflag{
		// 		proposeflag = true
		// 	}
		case req = <-cp.reqChan:
			cp.reqs.Reqs = append(cp.reqs.Reqs, req)
			cp.reqNum += cp.interval
			// if !makebatch {
			// 	makebatch = true
			// 	cp.getBatch()
			// }
		case content := <-cp.restoreChan:
			reqs := new(message.Batch)
			proto.Unmarshal(content, reqs)
			cp.reqs.Reqs = append(cp.reqs.Reqs, reqs.Reqs...)
			cp.reqNum += cp.interval * len(reqs.Reqs)
			// if !makebatch {
			// 	makebatch = true
			// 	cp.getBatch()
			// }
		case <-timer.C:
			f := cp.getBatch()
			if !f {
				timer.Reset(time.Millisecond * time.Duration(100))
			}
		case <-cp.tryPropose:
			cp.getBatch()
			// timer.Reset(time.Millisecond * time.Duration(100))
		}
	}
}
func (cp *ClientMsgProcessor) getBatch() bool {
	if cp.reqNum == 0 {
		return false
	}

	if cp.reqNum <= cp.batch {
		payloadBytes, _ := proto.Marshal(cp.reqs)
		cp.reqs.Reset()
		cp.startId += int32(cp.reqNum)
		fmt.Println("get batch 1: ", cp.reqNum, " ", len(payloadBytes))
		cp.reqNum = 0

		cp.reqc <- payloadBytes
	} else {
		reqs := cp.reqs.Reqs[0 : cp.batch/cp.interval]
		cp.reqs.Reqs = cp.reqs.Reqs[cp.batch/cp.interval:]
		clientreqs := new(message.Batch)
		clientreqs.Reqs = reqs
		payloadBytes, _ := proto.Marshal(clientreqs)
		cp.startId += int32(cp.batch)
		cp.reqNum -= cp.batch
		fmt.Println("get batch 2: ", cp.batch)
		cp.reqc <- payloadBytes
	}
	return true
}

func (cp *ClientMsgProcessor) startRpcServer() {
	rpc.Register(cp)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(cp.port))
	if err != nil {
		panic(err)
	}
	go http.Serve(listener, nil)
}

func (cp *ClientMsgProcessor) Request(req *message.ClientReq, resp *message.Response) error {
	if req.StartId == 1 {
		cp.interval = int(req.ReqNum)
	}
	cp.reqChan <- req
	// FOR TEST: reply directly
	// cli, err := rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(cp.port+100))
	// if err != nil {
	// 	panic(err)
	// }
	// arg := &message.NodeReply{
	// 	StartID: uint32(req.StartId),
	// 	ReqNum:  uint32(req.ReqNum),
	// }
	// rep := &message.Response{}
	// go cli.Call("Client.NodeFinish", arg, rep)
	return nil
}

func (cp *ClientMsgProcessor) ReplyServer() {
	// cli, err := rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(cp.port+100))
	var cli *rpc.Client
	var err error

	for {
		req := <-cp.repc
		if cli == nil {
			cli, err = rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(cp.port+100))
			if err != nil {
				panic(err)
			}
		}
		clientreqs := new(message.Batch)
		proto.Unmarshal(req, clientreqs)
		//reply to client
		for i := 0; i < len(clientreqs.Reqs); i++ {
			arg := &message.NodeReply{
				StartID: uint32(clientreqs.Reqs[i].StartId),
				ReqNum:  uint32(clientreqs.Reqs[i].ReqNum),
			}
			resp := &message.Response{}
			go cli.Call("Client.NodeFinish", arg, resp)
		}
	}
}
