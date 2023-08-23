package server

import (
	"fmt"
	"sync"
	"time"

	"bkr-go/crypto/bls"
	"bkr-go/transport"
	"bkr-go/transport/info"
	"bkr-go/transport/message"

	"go.uber.org/zap"
)

type asyncCommSubset struct {
	st            *state
	lg            *zap.Logger
	n             uint64
	thld          uint64
	sequence      uint64
	numDecided    uint64
	numFinished   uint64
	numDecidedOne uint64
	instances     []*instance
	proposer      *Proposer
	reqc          chan *message.ConsMessage
	startTime     int64
	zeroTime      int64
	endTime       int64
	lock          sync.Mutex
}

func initACS(st *state,
	lg *zap.Logger,
	tp transport.Transport,
	blsSig *bls.BlsSig,
	proposer *Proposer,
	seq uint64, n uint64,
	reqc chan *message.ConsMessage) *asyncCommSubset {
	re := &asyncCommSubset{
		st:        st,
		lg:        lg,
		proposer:  proposer,
		n:         n,
		sequence:  seq,
		instances: make([]*instance, n),
		reqc:      reqc,
		startTime: time.Now().UnixNano(),
		lock:      sync.Mutex{}}
	re.thld = 2*n/3 + 1
	for i := info.IDType(0); i < info.IDType(n); i++ {
		re.instances[i] = initInstance(lg, tp, blsSig, seq, n, re.thld, uint64(i))
	}
	return re
}

func (acs *asyncCommSubset) insertMsg(msg *message.ConsMessage) {
	isDecided, isFinished := acs.instances[msg.Proposer].insertMsg(msg)
	if isDecided {
		acs.lock.Lock()
		defer acs.lock.Unlock()

		if !acs.instances[msg.Proposer].decidedOne() && info.IDType(msg.Proposer) == acs.proposer.id {
			fmt.Printf("ID %d decided zero at %d\n", msg.Proposer, msg.Sequence)
		}
		acs.numDecided++
		// if acs.numDecided == 1 {
		// 	acs.proposer.proceed(acs.sequence)
		// }
		if acs.instances[msg.Proposer].decidedOne() {
			acs.numDecidedOne++
		}
		if acs.numDecidedOne == acs.thld {
			if acs.zeroTime == int64(0) {
				acs.zeroTime = time.Now().UnixNano()
			}
			for i, inst := range acs.instances {
				inst.canVoteZero(info.IDType(i), acs.sequence)
			}
		}
		if acs.numDecided == acs.n {
			for _, inst := range acs.instances {
				proposal := inst.getProposal()
				if inst.decidedOne() && len(proposal.Content) != 0 {
					inst.lg.Info("executed",
						zap.Int("proposer", int(proposal.Proposer)),
						zap.Int("seq", int(msg.Sequence)),
						zap.Int("content", int(proposal.Content[0])))
					acs.reqc <- proposal
				} else if info.IDType(proposal.Proposer) == acs.proposer.id && len(proposal.Content) != 0 {
					inst.lg.Info("repropose",
						zap.Int("proposer", int(proposal.Proposer)),
						zap.Int("seq", int(proposal.Sequence)),
						zap.Int("content", int(proposal.Content[0])))
					// acs.proposer.propose(proposal.Content)
					// restore the requests of proposal
					acs.proposer.restoreChan <- proposal.Content
				} else if inst.decidedOne() {
					inst.lg.Info("empty",
						zap.Int("proposer", int(proposal.Proposer)),
						zap.Int("seq", int(proposal.Sequence)))
				}
			}
			acs.endTime = time.Now().UnixNano()
			acs.lg.Info("ACS execute time",
				zap.Int("seq", int(acs.sequence)),
				zap.Int("startTime", int(acs.startTime)),
				zap.Int("VoteZero", int(acs.zeroTime-acs.startTime)),
				zap.Int("endtime", int(acs.endTime-acs.startTime)),
			)

			fmt.Println("acs propose: ", acs.sequence+1)
			acs.proposer.pc <- acs.sequence + 1
		}
	} else if isFinished {
		acs.lock.Lock()
		defer acs.lock.Unlock()

		acs.numFinished++
		if acs.numFinished == acs.n {
			acs.st.garbageCollect(acs.sequence)
		}
	}
}
