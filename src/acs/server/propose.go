package server

import (
	"fmt"
	"sync"

	"bkr-go/transport"
	"bkr-go/transport/info"
	"bkr-go/transport/message"

	"go.uber.org/zap"
)

// Proposer is responsible for proposing requests
type Proposer struct {
	lg           *zap.Logger
	reqc         chan []byte
	tp           transport.Transport
	seq          uint64
	id           info.IDType
	hasProposed  map[uint64]bool
	proposerChan chan struct{}
	pc           chan uint64
	restoreChan  chan []byte
	lock         sync.Mutex
}

func initProposer(lg *zap.Logger, tp transport.Transport, id info.IDType, reqc chan []byte, proposerChan chan struct{}, pc chan uint64, restoreChan chan []byte) *Proposer {
	hasProposed := make(map[uint64]bool)
	proposer := &Proposer{lg: lg, tp: tp, id: id, reqc: reqc, hasProposed: hasProposed, proposerChan: proposerChan, pc: pc, restoreChan: restoreChan, lock: sync.Mutex{}}
	go proposer.run()
	return proposer
}

func (proposer *Proposer) proceed(seq uint64) {
	proposer.lock.Lock()
	defer proposer.lock.Unlock()

	if proposer.seq <= seq {
		proposer.reqc <- []byte{} // insert an empty reqeust
	}
}

func (proposer *Proposer) run() {
	var req []byte
	var sequence uint64
	req = <-proposer.reqc
	proposer.propose(req, sequence)
	for {
		// req = <-proposer.reqc
		// proposer.propose(req)
		sequence = <-proposer.pc
		fmt.Println(sequence)
		if !proposer.hasProposed[sequence] {
			proposer.hasProposed[sequence] = true
			proposer.proposerChan <- struct{}{}
			req = <-proposer.reqc
			proposer.propose(req, sequence)
			// time.Sleep(time.Millisecond * time.Duration(200))
		}

	}
}

// Propose broadcast a propose message with the given request and the current sequence number
func (proposer *Proposer) propose(request []byte, seq uint64) {
	msg := &message.ConsMessage{
		Type:     message.ConsMessage_VAL,
		Proposer: uint32(proposer.id),
		From:     uint32(proposer.id),
		Sequence: seq,
		Content:  request}

	if len(request) > 0 {
		proposer.lg.Info("propose",
			zap.Int("proposer", int(msg.Proposer)),
			zap.Int("seq", int(msg.Sequence)),
			zap.Int("content", int(msg.Content[0])))
	}

	proposer.tp.Broadcast(msg)
}
