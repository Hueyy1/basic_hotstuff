package service

import (
	"hxy352/src/types"
)

type Pacemaker interface {
}

type PacemakerService struct {
	qcHigh *types.QuorumCert
}

func (s *PacemakerService) GetLeader() {

}

func (s *PacemakerService) UpdateQcHigh() {
	// 	  public:
	//    PMHighTail(int32_t parent_limit): parent_limit(parent_limit) {}
	//    void init() {
	//        hqc_tail = hsc->get_genesis();
	//        reg_hqc_update();
	//        reg_proposal();
	//        reg_receive_proposal();
	//    }
}

func (s *PacemakerService) OnBeat() {
	// virtual promise_t beat() = 0;
}

func (s *PacemakerService) OnNextSyncView() {

}

func (s *PacemakerService) OnReceiveNewView() {

}

func (s *PacemakerService) GetProposer() {
	// virtual promise_t beat() = 0;
	// /** Get the current proposer. */
}

func (s *PacemakerService) GetParents() {
	///	  ** Select the parent blocks for a new block.
	//     * @return Parent blocks. The block at index 0 is the direct parent, while
	//     * the others are uncles/aunts. The returned vector should be non-empty. */
	//    virtual std::vector<block_t> get_parents() = 0;
	// 对应b_leaf
}
