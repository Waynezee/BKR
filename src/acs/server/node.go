package server

import (
	"crypto/ecdsa"
	"runtime"

	"bkr-go/crypto/bls"
	"bkr-go/transport"
	"bkr-go/transport/http"
	"bkr-go/transport/info"

	"go.uber.org/zap"
)

// Node is a local process
type Node struct {
	reply     chan []byte
	proposer  *Proposer
	transport *transport.Transport
}

// InitNode initiate a node for processing messages
func InitNode(lg *zap.Logger, blsSig *bls.BlsSig, id info.IDType, n uint64, port int, peers []http.Peer, pk *ecdsa.PrivateKey, batch int) {
	tp, msgc, reqc, repc, proposeChan, restoreChan := transport.InitTransport(lg, id, port, peers, pk, batch)
	pc := make(chan uint64, 100)
	proposer := initProposer(lg, tp, id, reqc, proposeChan, pc, restoreChan)
	state := initState(lg, tp, blsSig, id, proposer, n, repc, pc)
	for i := 0; i < runtime.NumCPU()-1; i++ {
		go func() {
			for {
				msg := <-msgc
				state.insertMsg(msg)
			}
		}()
	}
	for {
		msg := <-msgc
		state.insertMsg(msg)
	}
}
