package service

//import (
//	"hxy352/src/log"
//	"hxy352/src/model"
//	"hxy352/src/types"
//)
//
//type QCManager struct {
//}
//
//// NewQCManager creates a new QcManager instance.
//func NewQCManager() *QCManager {
//	return &QCManager{}
//}
//
//// updateHighQC attempts to update the highQC, but does not verify the qc first.
//// This method is meant to be used instead of the exported UpdateHighQC internally
//// in this package when the qc has already been verified.
//func (q *QCManager) UpdateHighQC(blockChain model.BlockChain, highQC types.QuorumCert, qc types.QuorumCert) {
//	newBlock, ok := blockChain.Get(qc.BlockHash())
//	if !ok {
//		log.Info("updateHighQC: Could not find block referenced by new QC!")
//		return
//	}
//
//	if newBlock.View() > highQC.View() {
//		s.highQC = qc
//		log.Debug("HighQC updated")
//	}
//}
