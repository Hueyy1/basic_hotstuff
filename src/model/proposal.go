package model

import "hxy352/src/types"

// Proposal 结构体示例，细节根据实际定义调整
type Proposal struct {
	// 比如包含 Block、QC 等字段
	Block *Block
	QC    *types.QuorumCert
}
