package consensus

import (
	"context"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/commonpb"
	"hxy352/src/types"
	"sync"
	"time"
)

type BasicHotStuffWithCogsworth struct {
	*HotStuffImpl

	verifiedTimeoutVotes map[types.View][]types.QuorumSignature

	wishNextViewTmp        map[types.View][]types.QuorumSignature
	wishFinishedTmp        map[types.View]bool
	timeoutVoteFinishedTmp map[types.View]bool

	// pacemaker
	PaceMaker *CogsWorthPacemaker
}

func NewBasicHotStuffWithCogsworth(conf *model.ReplicaConf, gConf *model.Config) HotStuff {
	log.Infof("*** start init BasicHotStuffWithCogsworth ***")
	hs := &BasicHotStuffWithCogsworth{
		HotStuffImpl: NewHotStuffImpl(conf, gConf),

		verifiedTimeoutVotes: make(map[types.View][]types.QuorumSignature),

		wishNextViewTmp:        make(map[types.View][]types.QuorumSignature),
		wishFinishedTmp:        make(map[types.View]bool),
		timeoutVoteFinishedTmp: make(map[types.View]bool),
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

	//go hs.metric.Handle()

	hs.PaceMaker = NewCogsWorthPacemaker(hs)

	return hs
}

func (hs *BasicHotStuffWithCogsworth) Run() {
	go hs.HandleMsg()
	go hs.PaceMaker.OnBeat()

	go func() {
		cView := hs.CurrentView
		hs.ProcessCurrentViewQueue(cView)
		time.Sleep(10 * time.Millisecond)
	}()
}

func (hs *BasicHotStuffWithCogsworth) HandleMsg() {

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
			case commonpb.MessageType_WishNextView:
				hs.OnReceiveWishNextView(msg)

			case commonpb.MessageType_Timeout:
				hs.OnReceiveTimeout(msg)

			case commonpb.MessageType_TimeoutVote:
				hs.OnReceiveTimeoutVote(msg)

			case commonpb.MessageType_NewView:
				hs.OnReceiveNewView(msg)

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

			}

		case <-hs.timeout.Timeout():

			log.Warnf("Timeout received, go to new view !!!")

			hs.PaceMaker.WishToAdvance()

		}
	}

}

func (hs *BasicHotStuffWithCogsworth) ProcessCurrentViewQueue(cView types.View) {

	lastMsg := hs.MsgQueue.Pop(cView)

	if lastMsg == nil {
		log.Debugf("ProcessCurrentViewQueue: no msg found for view %d", cView)
		return
	}

	log.Infof("ProcessCurrentViewQueue: msg(%s) found for view %d, lastMsg: %+v", lastMsg.GetType(), cView, lastMsg)

	// todo: 除了decide，猜测其他情况下可能有bug，只处理了一个vote

	switch lastMsg.GetType() {
	case commonpb.MessageType_WishNextView:
		hs.OnReceiveWishNextView(lastMsg)

		for {
			tMsg := hs.MsgQueue.Pop(cView)
			if tMsg == nil {
				break
			}

			log.Infof("ProcessCurrentViewQueue: continue wish next view msg found for view %d", cView)
			hs.OnReceiveWishNextView(tMsg)
		}

	case commonpb.MessageType_Timeout:
		hs.OnReceiveTimeout(lastMsg)

		log.Infof("OnReceiveTimeout: msg found for view %d", cView)

	case commonpb.MessageType_TimeoutVote:
		hs.OnReceiveTimeoutVote(lastMsg)

		for {
			tMsg := hs.MsgQueue.Pop(cView)
			if tMsg != nil && tMsg.GetType() == commonpb.MessageType_TimeoutVote {
				log.Infof("OnReceiveTimeoutVote: continue msg found for view %d", cView)
				hs.OnReceiveTimeoutVote(tMsg)
			} else {
				break
			}
		}

	case commonpb.MessageType_NewView:
		hs.OnReceiveNewView(lastMsg)
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
		hs.ProcessCurrentViewQueue(cView + 1)
	}

}

// OnReceiveCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuffWithCogsworth) OnReceiveCommitVote(msg *basichotstuffpb.Msg) {
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

	if hs.finishedCommitVotes[pc.BlockHash()] {
		log.Infof("OnReceiveCommitVote: view %d votes have finished, ignore", msg.GetView())
		return
	}

	if !hs.crypto.VerifyPartialCert(block, pc) {
		log.Info("OnReceiveCommitVote: Vote could not be verified!")
		return
	}

	log.Debugf("OnReceiveCommitVote: Vote PC verified: %.8s", pc.BlockHash())

	//hs.mut.Lock()
	//defer hs.mut.Unlock()

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
	hs.finishedCommitVotes[pc.BlockHash()] = true

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
	log.Infof("OnReceiveCommitVote: exec cmd: %s %s", msg.GetBlock().Hash, block.Command())

	//// view number + 1
	//hs.CurrentView += 1

	// reset currentBlock
	cmd := string(hs.CurrentBlock.Command())
	hs.CurrentBlock = nil
	hs.CmdCache.Finish(cmd)

	// 不触发超时，直接进入new view
	hs.HighQC = qc
	//hs.PaceMaker.Ready <- struct{}{}

	//hs.mut.Unlock()

	//hs.mut.Lock()
	// send new view
	//hs.SendNewView()
	//hs.mut.Unlock()

	// todo: if
	// send response after new-view
	hs.SendResponse(cmd)

	hs.timeout.Stop()

	hs.PaceMaker.WishToAdvance()

}

// OnReceiveDecide is called to decide on a block.
func (hs *BasicHotStuffWithCogsworth) OnReceiveDecide(msg *basichotstuffpb.Msg) {
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

	//// view number + 1
	//hs.CurrentView = types.View(msg.View + 1)
	//log.Infof("OnReceiveDecide: set new view %d", hs.CurrentView)

	// reset block
	cmd := string(block.Command())
	hs.CurrentBlock = nil

	// leader start change new view
	//if hs.GetLeader() == hs.Conf.Id {
	//	hs.ViewChanging = true
	//}

	// todo: update highqc ? is it right?
	hs.HighQC = qc
	hs.CommitQC = qc

	//hs.PaceMaker.Ready <- struct{}{}

	// send response
	hs.SendResponse(cmd)

	hs.timeout.Stop()

	hs.mut.Unlock()

	// send wish to next leader
	hs.PaceMaker.WishToAdvance()

}

func (hs *BasicHotStuffWithCogsworth) SendWishNextView() {
	sig, _ := hs.crypto.Sign(hs.CurrentView.ToBytes())
	msg := &basichotstuffpb.Msg{
		Type:      commonpb.MessageType_WishNextView,
		View:      uint64(hs.CurrentView),
		ReplicaId: uint32(hs.Conf.Id),
		ViewSig:   basichotstuffpb.QuorumSignatureToProto(sig),
	}

	if hs.GetLeaderId() == hs.Conf.Id {
		hs.OnReceiveWishNextView(msg)
		return
	}

	hs.GetLeaderNode().WishNextView(context.Background(), msg)
}

func (hs *BasicHotStuffWithCogsworth) OnReceiveWishNextView(msg *basichotstuffpb.Msg) {
	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	newView := types.View(msg.GetView())

	log.Infof("OnReceiveWishNextView: view:%d", msg.GetView())

	if hs.wishFinishedTmp[newView] {
		log.Warnf("OnReceiveWishNextView: view:%d, wish finished , ignore", msg.GetView())
		return
	}

	hs.timeout.SoftStart()

	sig := basichotstuffpb.QuorumSignatureFromProto(msg.GetViewSig())

	if !hs.crypto.Verify(sig, newView.ToBytes()) {
		log.Errorf("OnReceiveWishNextView: invalid view quorum signature")
		return
	}

	hs.wishNextViewTmp[newView] = append(hs.wishNextViewTmp[newView], sig)

	length := len(hs.wishNextViewTmp[newView])

	log.Infof("OnReceiveWishNextView: view:%d, current length: %d", newView, length)

	// need f + 1 messages
	if length < hs.NodeManager.TimeoutQuorumSize {
		return
	}

	// create timeout cert
	cert, err := hs.crypto.CreateTimeoutCert(newView, hs.wishNextViewTmp[newView])
	if err != nil {
		log.Errorf("OnReceiveWishNextView: could not create timeout cert")
		return
	}

	timeoutMsg := &basichotstuffpb.Msg{
		Type:      commonpb.MessageType_Timeout,
		View:      msg.GetView(),
		TC:        basichotstuffpb.TimeoutCertToProto(cert),
		ReplicaId: uint32(hs.Conf.Id),
	}
	hs.NodeManager.GetNodesCfg().Timeout(context.Background(), timeoutMsg)
	log.Infof("OnReceiveWishNextView: send timeout(%d) msg ", newView)

	// clean wishes
	delete(hs.wishNextViewTmp, newView)
	hs.wishFinishedTmp[newView] = true

	// vote myself
	tv := types.NewTimeoutVote(newView, cert)
	sigVote, _ := hs.crypto.CreateTimeoutVoteCert(tv)
	hs.OnReceiveTimeoutVote(&basichotstuffpb.Msg{
		Type:      commonpb.MessageType_TimeoutVote,
		View:      msg.GetView(),
		ReplicaId: uint32(hs.Conf.Id),
		ViewSig:   basichotstuffpb.QuorumSignatureToProto(sigVote),
		TC:        timeoutMsg.TC,

		// todo: 此处为QC？？
		QC: basichotstuffpb.QuorumCertToProto(hs.PrepareQC),
	})

}

func (hs *BasicHotStuffWithCogsworth) OnReceiveTimeout(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveTimeout: view:%d", msg.GetView())

	if msg.GetType() != commonpb.MessageType_Timeout {
		return
	}

	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	tc := basichotstuffpb.TimeoutCertFromProto(msg.GetTC())
	ok := hs.crypto.VerifyTimeoutCert(tc)
	if !ok {
		log.Errorf("OnReceiveTimeout: invalid timeout cert")
		return
	}

	newView := types.View(msg.GetView())
	tv := types.NewTimeoutVote(newView, tc)
	sig, _ := hs.crypto.CreateTimeoutVoteCert(tv)

	hs.GetLeaderNode().TimeoutVote(context.Background(), &basichotstuffpb.Msg{
		Type:      commonpb.MessageType_TimeoutVote,
		View:      msg.GetView(),
		ReplicaId: uint32(hs.Conf.Id),
		ViewSig:   basichotstuffpb.QuorumSignatureToProto(sig),
		TC:        msg.GetTC(),

		// todo: 此处为QC？？
		QC: basichotstuffpb.QuorumCertToProto(hs.PrepareQC),
	})

	log.Infof("OnReceiveTimeout: send vote view:%d", msg.GetView())

}

func (hs *BasicHotStuffWithCogsworth) OnReceiveTimeoutVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveTimeoutVote: view:%d", msg.GetView())

	if msg.GetType() != commonpb.MessageType_TimeoutVote {
		log.Warnf("OnReceiveTimeoutVote: view:%s, wrong type ", msg.GetType())
		return
	}

	newView := types.View(msg.GetView())

	if hs.timeoutVoteFinishedTmp[newView] != true {
		hs.timeout.SoftStart()
	} else {
		log.Warnf("OnReceiveTimeoutVote: view:%s, vote finished, ingore ", msg.GetType())
		return
	}

	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	sig := basichotstuffpb.QuorumSignatureFromProto(msg.GetViewSig())

	tv := types.NewTimeoutVote(newView, basichotstuffpb.TimeoutCertFromProto(msg.GetTC()))

	if !hs.crypto.VerifyTimeoutVoteCert(tv, sig) {
		log.Errorf("OnReceiveTimeoutVote: invalid view quorum signature")
		return
	}

	hs.verifiedTimeoutVotes[newView] = append(hs.verifiedTimeoutVotes[newView], sig)

	hs.highQCTmp[newView] = append(hs.highQCTmp[newView], basichotstuffpb.QuorumCertFromProto(msg.GetQC()))

	length := len(hs.verifiedTimeoutVotes[newView])

	hs.ViewChanging = true

	log.Infof("OnReceiveTimeoutVote: get view %d votes length %d ", newView, length)

	if length < hs.NodeManager.QuorumSize {
		return
	}

	qc, _ := hs.crypto.CreateTimeoutQuorumCert(tv, hs.verifiedTimeoutVotes[newView])
	hs.processNewView(msg, qc)

	delete(hs.verifiedTimeoutVotes, newView)
}

// OnReceiveNewView is called when a new view message is received.
func (hs *BasicHotStuffWithCogsworth) OnReceiveNewView(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveNewView: receive new view: %d, msg: %+v", msg.GetView(), msg)

	tv := types.NewTimeoutVote(types.View(msg.View), basichotstuffpb.TimeoutCertFromProto(msg.GetTC()))
	qc := basichotstuffpb.QuorumCertFromProto(msg.GetQC())
	if !hs.crypto.VerifyTimeoutQuorumCert(tv, qc) {
		log.Errorf("OnReceiveNewView: invalid view quorum cert")
		return
	}

	hs.processNewView(msg, qc)
}

func (hs *BasicHotStuffWithCogsworth) SendNewView() {

}

func (hs *BasicHotStuffWithCogsworth) processNewView(msg *basichotstuffpb.Msg, qc types.QuorumCert) {

	if hs.GetLeaderId() == hs.Conf.Id {
		// todo: maybe bug block not found ?????
		block, _ := hs.BlockChain.Get(hs.HighQC.BlockHash())
		hs.NodeManager.GetNodesCfg().NewView(context.Background(), &basichotstuffpb.Msg{
			Type:      commonpb.MessageType_NewView,
			View:      msg.GetView(),
			QC:        basichotstuffpb.QuorumCertToProto(qc),
			ReplicaId: uint32(hs.Conf.Id),
			TC:        msg.GetTC(),

			Block: basichotstuffpb.BlockToProto(block),
		})
	}

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

	hs.timeout.Stop()

	hs.timeoutVoteFinishedTmp[hs.CurrentView] = true

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
