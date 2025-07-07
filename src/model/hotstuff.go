package model

import (
	"go.uber.org/zap"
)

type HotStuff struct {
	//blockChain modules.BlockChain
	logger zap.Logger

	// protocol variables

	bLock *Block // the currently locked block
}
