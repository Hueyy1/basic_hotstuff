package consensus

type HotStuff interface {
	Msg(msgType pb.MsgType, node *pb.Block, qc *pb.QuorumCert) *pb.Msg
	VoteMsg(msgType pb.MsgType, node *pb.Block, qc *pb.QuorumCert, justify []byte) *pb.Msg
	CreateLeaf(parentHash []byte, cmds []string, justify *pb.QuorumCert) *pb.Block
	QC(msgType pb.MsgType, sig tcrsa.Signature, blockHash []byte) *pb.QuorumCert
	MatchingMsg(msg *pb.Msg, msgType pb.MsgType) bool
	MatchingQC(qc *pb.QuorumCert, msgType pb.MsgType) bool
	SafeNode(node *pb.Block, qc *pb.QuorumCert) bool
	GetMsgEntrance() chan<- *pb.Msg
	GetSelfInfo() *config.ReplicaInfo
	SafeExit()
}
