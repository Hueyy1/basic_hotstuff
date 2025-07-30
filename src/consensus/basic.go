package consensus

import (
	"context"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/commonpb"
	"hxy352/src/types"
	"strconv"
	"sync"
	"time"
)

type BasicHotStuff struct {
	*HotStuffImpl
}

func NewBasicHotStuff(conf *model.ReplicaConf, gConf *model.Config) HotStuff {
	log.Infof("*** start init BasicHotStuff ***")

	hs := &BasicHotStuff{
		HotStuffImpl: NewHotStuffImpl(conf, gConf),
	}

	var err error
	hs.HighQC, err = hs.crypto.CreateQuorumCert(model.GetGenesis(), []types.PartialCert{})
	if err != nil {
		log.Panicf("Failed to create initial quorum certificate: %v", err)
	}
	hs.PreCommitQC = hs.HighQC

	hs.ViewChangeCond = sync.NewCond(&hs.mut)

	go func() {
		if hs.GetLeaderId() == hs.Conf.Id {
			hs.Ready <- struct{}{}
		}
	}()

	return hs
}

func (hs *BasicHotStuff) Run() {
	go hs.HandleMsg()
	go hs.HandleReq()

	go func() {
		cView := hs.CurrentView
		hs.ProcessCurrentViewQueue(cView)
		time.Sleep(10 * time.Millisecond)
	}()
}

func (hs *BasicHotStuff) HandleMsg() {

	for {
		select {
		case msg := <-hs.MsgChan:

			//// try process current view msg
			//hs.ProcessCurrentViewQueue(hs.CurrentView)

			//hs.mut.Lock()
			cView := hs.CurrentView
			//hs.mut.Unlock()
			msgView := types.View(msg.GetView())

			if msgView > cView {
				hs.MsgQueue.Put(msg)
				log.Warnf("HandleMsg: msg(%s) %d too fast, current view: %d", msg.GetType(), msgView, cView)

				// try process current view msg
				hs.ProcessCurrentViewQueue(cView)

				continue
			} else if msgView < cView {
				log.Warnf("HandleMsg: msg(%s) %d too old, current view: %d", msg.GetType(), msgView, cView)

				// try process current view msg
				hs.ProcessCurrentViewQueue(cView)

				continue
			}

			// try process current view msg
			hs.ProcessCurrentViewQueue(cView)

			switch msg.Type {

			case commonpb.MessageType_Prepare:
				hs.OnReceivePrepare(msg)

			case commonpb.MessageType_PrepareVote:
				hs.OnReceivePrepareVote(msg)

			case commonpb.MessageType_PreCommit:
				hs.OnReceivePreCommit(msg)

			case commonpb.MessageType_PreCommitVote:
				hs.OnReceivePreCommitVote(msg)

			case commonpb.MessageType_Commit:
				hs.OnReceiveCommit(msg)

			case commonpb.MessageType_CommitVote:
				hs.OnReceiveCommitVote(msg)

			case commonpb.MessageType_Decide:
				hs.OnReceiveDecide(msg)

			case commonpb.MessageType_NewView:
				hs.OnReceiveNewView(msg)

			}

		case <-hs.timeout.Timeout():
			log.Warnf("Timeout received, go to new view !!!")

			// todo: timeout * 2
			// hs.timeout = service.NewTimeoutService()
			hs.timeout.Reset()
			hs.timeout.Stop()

			// todo: if need create empty block???
			//hs.BlockChain.Store(hs.CreateLeaf(hs.CurrentBlock.Parent(), types.QuorumCert{}, ""))

			hs.mut.Lock()

			hs.CurrentView += 1

			hs.CurrentBlock = nil

			// send new view msg
			hs.SendNewView()

			hs.mut.Unlock()

		}
	}

}

func (hs *BasicHotStuff) HandleReq() {
	for {
		select {
		case <-hs.Ready:

			for {
				req, ok := hs.CmdCache.Dequeue()
				log.Debugf("hs.CmdCache.Dequeue(): %+v", req)
				if !ok {
					continue
				}
				if hs.CmdCache.IsFinished(req.GetCmd()) {
					log.Debugf("hs.CmdCache.IsFinished(): %+v", req.GetCmd())
					continue
				}
				hs.SendPrepare(req)
				break
			}

		}
	}
}

func (hs *BasicHotStuff) ProcessCurrentViewQueue(cView types.View) {

	lastMsg := hs.MsgQueue.Pop(cView)

	if lastMsg == nil {
		log.Debugf("ProcessCurrentViewQueue: no msg found for view %d", cView)
		return
	}

	log.Infof("ProcessCurrentViewQueue: msg(%s) found for view %d, lastMsg: %+v", lastMsg.GetType(), cView, lastMsg)

	// todo: 除了decide，猜测其他情况下可能有bug，只处理了一个vote

	switch lastMsg.GetType() {
	case commonpb.MessageType_NewView:

		// leader
		hs.OnReceiveNewView(lastMsg)

		for {
			tMsg := hs.MsgQueue.Pop(cView)
			if tMsg == nil {
				break
			}

			log.Infof("ProcessCurrentViewQueue: continue new view msg found for view %d", cView)
			hs.OnReceiveNewView(tMsg)
		}

	case commonpb.MessageType_Prepare:
		hs.OnReceivePrepare(lastMsg)
	case commonpb.MessageType_PrepareVote:
		hs.OnReceivePrepareVote(lastMsg)
	case commonpb.MessageType_PreCommit:
		hs.OnReceivePreCommit(lastMsg)
	case commonpb.MessageType_PreCommitVote:
		hs.OnReceivePreCommitVote(lastMsg)
	case commonpb.MessageType_Commit:
		hs.OnReceiveCommit(lastMsg)
	case commonpb.MessageType_CommitVote:
		hs.OnReceiveCommitVote(lastMsg)
	case commonpb.MessageType_Decide:
		hs.OnReceiveDecide(lastMsg)
		hs.ProcessCurrentViewQueue(hs.CurrentView)
	}

}

// OnReceiveCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuff) OnReceiveCommitVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveCommitVote: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_CommitVote) {
		log.Errorf("OnReceiveCommitVote: Commit vote msg does not match")
		return
	}

	pcPb := msg.GetPartialCert()
	if pcPb == nil {
		log.Errorf("OnReceiveCommitVote: partial certificate is nil")
		return
	}

	pc := basichotstuffpb.PartialCertFromProto(pcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	if !hs.crypto.VerifyPartialCert(block, pc) {
		log.Info("OnReceiveCommitVote: Vote could not be verified!")
		return
	}

	log.Debugf("OnReceiveCommitVote: Vote PC verified: %.8s", pc.BlockHash())

	hs.mut.Lock()
	defer hs.mut.Unlock()

	hs.CurrentBlock = block

	// store vote
	votes := hs.verifiedCommitVotes[pc.BlockHash()]
	votes = append(votes, pc)
	hs.verifiedCommitVotes[pc.BlockHash()] = votes

	if len(votes) < hs.NodeManager.QuorumSize {
		//hs.mut.Unlock()
		return
	}

	log.Debugf("OnReceiveCommitVote: get vote size: %d", len(votes))

	qc, err := hs.crypto.CreateQuorumCert(block, votes)
	if err != nil {
		log.Info("OnReceiveCommitVote: could not create QC for block: ", err)
		//hs.mut.Unlock()
		return
	}

	// store qc
	hs.CommitQC = qc

	// clean votes after create QC
	delete(hs.verifiedCommitVotes, pc.BlockHash())

	// store block
	hs.BlockChain.Store(block)

	// clean block
	hs.BlockChain.Clean(block)

	// send decide

	decideMsg := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_Decide,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(qc),
		ReplicaId:   uint32(hs.Conf.Id),
	}

	hs.NodeManager.GetNodesCfg().Decide(context.Background(), decideMsg)

	// exec cmd
	cmdInt, _ := strconv.Atoi(string(block.Command()))
	log.Infof("OnReceiveCommitVote: exec cmd: %s %d", msg.GetBlock().Hash, cmdInt)

	// view number + 1
	hs.CurrentView += 1

	// reset currentBlock
	cmd := string(hs.CurrentBlock.Command())
	hs.CurrentBlock = nil
	hs.CmdCache.Finish(cmd)

	//hs.mut.Unlock()

	//hs.mut.Lock()
	// send new view
	hs.SendNewView()
	//hs.mut.Unlock()

	// todo: if
	// send response after new-view
	hs.SendResponse(cmd)

	hs.timeout.Stop()

}

// OnReceiveDecide is called to decide on a block.
func (hs *BasicHotStuff) OnReceiveDecide(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveDecide: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Decide) {
		log.Errorf("OnReceiveDecide: msg does not match")
		return
	}

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceivePreCommit: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	if !hs.crypto.VerifyQuorumCert(block, qc) {
		log.Warnf("OnReceiveDecide: invalid quorum certificate for block: %s", msg.GetType().String())
		return
	}

	//if hs.CurrentBlock.Hash() != qc.BlockHash() {
	//	log.Warnf("OnReceiveDecide: Could not find block for vote: %.8s.", qc.BlockHash())
	//	return
	//}

	// Ensure the block is proposed by the expected leader
	leader := hs.GetLeaderId()
	if leader != block.Proposer() {
		log.Warnf("OnReceiveDecide: block was not proposed by the expected leader: %d, got: %d", leader, block.Proposer())
		return
	}

	// exec cmd
	log.Infof("OnReceiveDecide: exec cmd: %s %s", msg.GetBlock().Hash, block.Command())

	hs.CmdCache.Finish(string(block.Command()))

	// store block
	hs.BlockChain.Store(block)

	// clean block
	hs.BlockChain.Clean(block)

	hs.mut.Lock()
	defer hs.mut.Unlock()

	// view number + 1
	hs.CurrentView = types.View(msg.View + 1)
	log.Infof("OnReceiveDecide: set new view %d", hs.CurrentView)

	// reset block
	cmd := string(block.Command())
	hs.CurrentBlock = nil

	// leader start change new view
	if hs.GetLeaderId() == hs.Conf.Id {
		hs.ViewChanging = true
	}

	// todo: update highqc ? is it right?
	hs.HighQC = qc
	hs.CommitQC = qc

	// send new view to next leader
	hs.SendNewView()

	// send response
	hs.SendResponse(cmd)

	hs.timeout.Stop()

}

func (hs *BasicHotStuff) SendNewView() {
	// send new view to next leader

	l := hs.GetLeaderId()
	if l == hs.Conf.Id {
		// new view to myself

		//hs.mut.Lock()

		//hs.highQCTmp = append(hs.highQCTmp, hs.PrepareQC)
		hs.highQCTmp[hs.CurrentView] = append(hs.highQCTmp[hs.CurrentView], hs.PrepareQC)

		hs.ViewChanging = true

		hs.ProcessCurrentViewQueue(hs.CurrentView)
		//hs.ProcessCurrentViewQueue(hs.CurrentView + 1)

		length := len(hs.highQCTmp[hs.CurrentView])
		//log.Infof("highQCTmp: %v", hs.highQCTmp[hs.CurrentView])
		log.Infof("SendNewView: send new view %d to myself, View Changing... get %d qc", hs.CurrentView, length)

		if length < hs.NodeManager.QuorumSize {
			return
		}

		hs.processNewView()

		//hs.ProcessQueue(hs.CurrentView)

		//hs.mut.Unlock()
		return
	}

	log.Infof("OnSendNewView: sending new view to leader: %d", l)

	hs.GetLeaderNode().NewViewBasic(context.Background(), &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_NewView,
		View:        uint64(hs.CurrentView),
		Block:       nil,
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(hs.PrepareQC),
	})
	//hs.Ready <- struct{}{}
}

// OnReceiveNewView is called when a new view message is received.
func (hs *BasicHotStuff) OnReceiveNewView(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveNewView: receive new view: %d", msg.GetView())

	if hs.finishedViewChange[hs.CurrentView] == true {
		log.Infof("OnReceiveNewView: already finished view change: %d", msg.GetView())
		return
	}

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceiveNewView: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)

	t := hs.highQCTmp[hs.CurrentView]
	t = append(t, qc)
	hs.highQCTmp[hs.CurrentView] = t

	hs.ViewChanging = true
	//log.Infof("highQCTmp: %v", hs.highQCTmp[hs.CurrentView])
	log.Infof("OnReceiveNewView: View Changing... get %d qc", len(hs.highQCTmp[hs.CurrentView]))

	if len(hs.highQCTmp[hs.CurrentView]) < hs.NodeManager.QuorumSize {
		return
	}

	hs.processNewView()
}

func (hs *BasicHotStuff) processNewView() {

	maxQC := hs.PrepareQC

	for _, tmp := range hs.highQCTmp[hs.CurrentView] {
		if tmp.View() > maxQC.View() {
			maxQC = tmp
		}
	}

	hs.HighQC = maxQC

	// clean hs.highQCTmp
	delete(hs.highQCTmp, hs.CurrentView)

	// clean votes
	cleanFunc := func(m map[types.Hash][]types.PartialCert) {
		for k := range m {
			delete(m, k)
		}
	}
	cleanFunc(hs.verifiedPrepareVotes)
	cleanFunc(hs.verifiedPreCommitVotes)
	cleanFunc(hs.verifiedCommitVotes)
	log.Debug("processNewView: clean all votes")

	hs.finishedViewChange[hs.CurrentView] = true

	hs.ViewChanging = false
	log.Infof("processNewView: new view %d finished, next view len: %d", hs.CurrentView, len(hs.MsgQueue.Get(hs.CurrentView+1)))

	// 通知handleReq
	hs.Ready <- struct{}{}

	//// try to chase new msg
	//cView := hs.CurrentView
	//hs.ProcessQueue(cView)
	//
	//if hs.MsgQueue.Get(cView+1) != nil && len(hs.MsgQueue.Get(cView+1)) != 0 {
	//	// continue process Queue
	//	log.Infof("processNewView: continue processNewView: %d", cView+1)
	//	hs.ProcessQueue(cView + 1)
	//} else {
	//	log.Infof("processNewView: broadcast view %d change finished", cView)
	//	hs.ViewChangeCond.Broadcast() // 唤醒等待请求
	//}

}
