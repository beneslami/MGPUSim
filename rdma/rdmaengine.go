// Package rdma provides the implementation of an RDMA engine.
package rdma

import (
	"log"
	"reflect"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/cache"
	"gitlab.com/akita/util/tracing"
)

type transaction struct {
	fromInside  akita.Msg
	fromOutside akita.Msg
	toInside    akita.Msg
	toOutside   akita.Msg
}

// An Engine is a component that helps one GPU to access the memory on
// another GPU
type Engine struct {
	*akita.TickingComponent

	ToOutside akita.Port

	ToL1 akita.Port
	ToL2 akita.Port

	CtrlPort akita.Port

	isDraining              bool
	pauseIncomingReqsFromL1 bool
	currentDrainReq         *DrainReq

	localModules           cache.LowModuleFinder
	RemoteRDMAAddressTable cache.LowModuleFinder

	transactionsFromOutside []transaction
	transactionsFromInside  []transaction
}

// SetLocalModuleFinder sets the table to lookup for local data.
func (e *Engine) SetLocalModuleFinder(lmf cache.LowModuleFinder) {
	e.localModules = lmf
}

// Tick checks if make progress
func (e *Engine) Tick(now akita.VTimeInSec) bool {
	madeProgress := false

	madeProgress = e.processFromCtrlPort(now) || madeProgress
	if e.isDraining {
		madeProgress = e.drainRDMA(now) || madeProgress
	}
	madeProgress = e.processFromL1(now) || madeProgress
	madeProgress = e.processFromL2(now) || madeProgress
	madeProgress = e.processFromOutside(now) || madeProgress

	return madeProgress
}

func (e *Engine) processFromCtrlPort(now akita.VTimeInSec) bool {
	req := e.CtrlPort.Peek()
	if req == nil {
		return false
	}

	req = e.CtrlPort.Retrieve(now)
	switch req := req.(type) {
	case *DrainReq:
		e.currentDrainReq = req
		e.isDraining = true
		e.pauseIncomingReqsFromL1 = true
		return true
	case *RestartReq:
		return e.processRDMARestartReq(now)
	default:
		log.Panicf("cannot process request of type %s", reflect.TypeOf(req))
		return false
	}
}

func (e *Engine) processRDMARestartReq(now akita.VTimeInSec) bool {
	restartCompleteRsp := RestartRspBuilder{}.
		WithSendTime(now).
		WithSrc(e.CtrlPort).
		WithDst(e.currentDrainReq.Src).
		Build()
	err := e.CtrlPort.Send(restartCompleteRsp)

	if err != nil {
		return false
	}
	e.currentDrainReq = nil
	e.pauseIncomingReqsFromL1 = false

	return true
}

func (e *Engine) drainRDMA(now akita.VTimeInSec) bool {
	if e.fullyDrained() {
		drainCompleteRsp := DrainRspBuilder{}.
			WithSendTime(now).
			WithSrc(e.CtrlPort).
			WithDst(e.currentDrainReq.Src).
			Build()

		err := e.CtrlPort.Send(drainCompleteRsp)
		if err != nil {
			return false
		}
		e.isDraining = false
		return true
	}
	return false
}

func (e *Engine) fullyDrained() bool {
	return len(e.transactionsFromOutside) == 0 &&
		len(e.transactionsFromInside) == 0
}

func (e *Engine) processFromL1(now akita.VTimeInSec) bool {
	if e.pauseIncomingReqsFromL1 {
		return false
	}

	req := e.ToL1.Peek()
	if req == nil {
		return false
	}

	switch req := req.(type) {
	case mem.AccessReq:
		return e.processReqFromL1(now, req)
	default:
		log.Panicf("cannot process request of type %s", reflect.TypeOf(req))
		return false
	}
}

func (e *Engine) processFromL2(now akita.VTimeInSec) bool {
	req := e.ToL2.Peek()
	if req == nil {
		return false
	}

	switch req := req.(type) {
	case mem.AccessRsp:
		return e.processRspFromL2(now, req)
	default:
		panic("unknown req type")
	}
}

func (e *Engine) processFromOutside(now akita.VTimeInSec) bool {
	req := e.ToOutside.Peek()
	if req == nil {
		return false
	}

	switch req := req.(type) {
	case mem.AccessReq:
		return e.processReqFromOutside(now, req)
	case mem.AccessRsp:
		return e.processRspFromOutside(now, req)
	default:
		log.Panicf("cannot process request of type %s", reflect.TypeOf(req))
		return false
	}
}

func (e *Engine) processReqFromL1(
	now akita.VTimeInSec,
	req mem.AccessReq,
) bool {
	dst := e.RemoteRDMAAddressTable.Find(req.GetAddress())

	cloned := e.cloneReq(req)
	cloned.Meta().Src = e.ToOutside
	cloned.Meta().Dst = dst
	cloned.Meta().SendTime = now

	err := e.ToOutside.Send(cloned)
	if err == nil {
		e.ToL1.Retrieve(now)

		tracing.TraceReqReceive(req, now, e)
		tracing.TraceReqInitiate(cloned, now, e,
			tracing.MsgIDAtReceiver(req, e))

		//fmt.Printf("%s req inside %s -> outside %s\n",
		//e.Name(), req.GetID(), cloned.GetID())

		trans := transaction{
			fromInside: req,
			toOutside:  cloned,
		}
		e.transactionsFromInside = append(e.transactionsFromInside, trans)

		return true
	}

	return false
}

func (e *Engine) processReqFromOutside(
	now akita.VTimeInSec,
	req mem.AccessReq,
) bool {
	dst := e.localModules.Find(req.GetAddress())

	cloned := e.cloneReq(req)
	cloned.Meta().Src = e.ToL2
	cloned.Meta().Dst = dst
	cloned.Meta().SendTime = now

	err := e.ToL2.Send(cloned)
	if err == nil {
		e.ToOutside.Retrieve(now)

		tracing.TraceReqReceive(req, now, e)
		tracing.TraceReqInitiate(cloned, now, e,
			tracing.MsgIDAtReceiver(req, e))

		//fmt.Printf("%s req outside %s -> inside %s\n",
		//e.Name(), req.GetID(), cloned.GetID())

		trans := transaction{
			fromOutside: req,
			toInside:    cloned,
		}
		e.transactionsFromOutside =
			append(e.transactionsFromOutside, trans)
		return true
	}
	return false
}

func (e *Engine) processRspFromL2(
	now akita.VTimeInSec,
	rsp mem.AccessRsp,
) bool {
	transactionIndex := e.findTransactionByRspToID(
		rsp.GetRespondTo(), e.transactionsFromOutside)
	trans := e.transactionsFromOutside[transactionIndex]

	rspToOutside := e.cloneRsp(rsp, trans.fromOutside.Meta().ID)
	rspToOutside.Meta().SendTime = now
	rspToOutside.Meta().Src = e.ToOutside
	rspToOutside.Meta().Dst = trans.fromOutside.Meta().Src

	err := e.ToOutside.Send(rspToOutside)
	if err == nil {
		e.ToL2.Retrieve(now)

		//fmt.Printf("%s rsp inside %s -> outside %s\n",
		//e.Name(), rsp.GetID(), rspToOutside.GetID())

		tracing.TraceReqFinalize(trans.toInside, now, e)
		tracing.TraceReqComplete(trans.fromOutside, now, e)

		e.transactionsFromOutside =
			append(e.transactionsFromOutside[:transactionIndex],
				e.transactionsFromOutside[transactionIndex+1:]...)
		return true
	}
	return false
}

func (e *Engine) processRspFromOutside(
	now akita.VTimeInSec,
	rsp mem.AccessRsp,
) bool {
	transactionIndex := e.findTransactionByRspToID(
		rsp.GetRespondTo(), e.transactionsFromInside)
	trans := e.transactionsFromInside[transactionIndex]

	rspToInside := e.cloneRsp(rsp, trans.fromInside.Meta().ID)
	rspToInside.Meta().SendTime = now
	rspToInside.Meta().Src = e.ToL2
	rspToInside.Meta().Dst = trans.fromInside.Meta().Src

	err := e.ToL2.Send(rspToInside)
	if err == nil {
		e.ToOutside.Retrieve(now)

		tracing.TraceReqFinalize(trans.toOutside, now, e)
		tracing.TraceReqComplete(trans.fromInside, now, e)

		//fmt.Printf("%s rsp outside %s -> inside %s\n",
		//e.Name(), rsp.GetID(), rspToInside.GetID())

		e.transactionsFromInside =
			append(e.transactionsFromInside[:transactionIndex],
				e.transactionsFromInside[transactionIndex+1:]...)

		return true
	}

	return false
}

func (e *Engine) findTransactionByRspToID(
	rspTo string,
	transactions []transaction,
) int {
	for i, trans := range transactions {
		if trans.toOutside != nil && trans.toOutside.Meta().ID == rspTo {
			return i
		}

		if trans.toInside != nil && trans.toInside.Meta().ID == rspTo {
			return i
		}
	}

	log.Panicf("transaction %s not found", rspTo)
	return 0
}

func (e *Engine) cloneReq(origin mem.AccessReq) mem.AccessReq {
	switch origin := origin.(type) {
	case *mem.ReadReq:
		read := mem.ReadReqBuilder{}.
			WithSendTime(origin.SendTime).
			WithSrc(origin.Src).
			WithDst(origin.Dst).
			WithAddress(origin.Address).
			WithByteSize(origin.AccessByteSize).
			Build()
		return read
	case *mem.WriteReq:
		write := mem.WriteReqBuilder{}.
			WithSendTime(origin.SendTime).
			WithSrc(origin.Src).
			WithDst(origin.Dst).
			WithAddress(origin.Address).
			WithData(origin.Data).
			WithDirtyMask(origin.DirtyMask).
			Build()
		return write
	default:
		log.Panicf("cannot clone request of type %s",
			reflect.TypeOf(origin))
	}
	return nil
}

func (e *Engine) cloneRsp(origin mem.AccessRsp, rspTo string) mem.AccessRsp {
	switch origin := origin.(type) {
	case *mem.DataReadyRsp:
		rsp := mem.DataReadyRspBuilder{}.
			WithSendTime(origin.SendTime).
			WithSrc(origin.Src).
			WithDst(origin.Dst).
			WithRspTo(rspTo).
			WithData(origin.Data).
			Build()
		return rsp
	case *mem.WriteDoneRsp:
		rsp := mem.WriteDoneRspBuilder{}.
			WithSendTime(origin.SendTime).
			WithSrc(origin.Src).
			WithDst(origin.Dst).
			WithRspTo(rspTo).
			Build()
		return rsp
	default:
		log.Panicf("cannot clone request of type %s",
			reflect.TypeOf(origin))
	}
	return nil
}

// SetFreq sets freq
func (e *Engine) SetFreq(freq akita.Freq) {
	e.TickingComponent.Freq = freq
}

// NewEngine creates new engine
func NewEngine(
	name string,
	engine akita.Engine,
	localModules cache.LowModuleFinder,
	remoteModules cache.LowModuleFinder,
) *Engine {
	e := new(Engine)
	e.TickingComponent = akita.NewTickingComponent(name, engine, 1*akita.GHz, e)
	e.localModules = localModules
	e.RemoteRDMAAddressTable = remoteModules

	e.ToL1 = akita.NewLimitNumMsgPort(e, 1, name+".ToL1")
	e.ToL2 = akita.NewLimitNumMsgPort(e, 1, name+".ToL2")
	e.CtrlPort = akita.NewLimitNumMsgPort(e, 1, name+".CtrlPort")
	e.ToOutside = akita.NewLimitNumMsgPort(e, 1, name+".ToOutside")

	return e
}
