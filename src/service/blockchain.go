package service

import (
	"hxy352/src/model"
	"hxy352/src/types"
	"sync"
)

// blockChain stores a limited amount of blocks in a map.
// blocks are evicted in LRU order.
type blockChain struct {
	mut           sync.Mutex
	pruneHeight   types.View
	blocks        map[types.Hash]*model.Block
	blockAtHeight map[types.View]*model.Block
	//pendingFetch  map[types.Hash]context.CancelFunc // allows a pending fetch operation to be canceled
}

// New creates a new blockChain with a maximum size.
// Blocks are dropped in least recently used order.
func NewBlockChain() model.BlockChain {
	bc := &blockChain{
		blocks:        make(map[types.Hash]*model.Block),
		blockAtHeight: make(map[types.View]*model.Block),
		//pendingFetch:  make(map[types.Hash]context.CancelFunc),
	}
	bc.Store(model.GetGenesis())
	return bc
}

// Store stores a block in the blockchain
func (chain *blockChain) Store(block *model.Block) {
	chain.mut.Lock()
	defer chain.mut.Unlock()

	chain.blocks[block.Hash()] = block
	chain.blockAtHeight[block.View()] = block

	//// cancel any pending fetch operations
	//if cancel, ok := chain.pendingFetch[block.Hash()]; ok {
	//	cancel()
	//}
}

// Get retrieves a block given its hash. It will only try the local cache.
func (chain *blockChain) LocalGet(hash types.Hash) (*model.Block, bool) {
	chain.mut.Lock()
	defer chain.mut.Unlock()

	block, ok := chain.blocks[hash]
	if !ok {
		return nil, false
	}

	return block, true
}

// Get retrieves a block given its hash. Get will try to find the block locally.
// If it is not available locally, it will try to fetch the block.
func (chain *blockChain) Get(hash types.Hash) (block *model.Block, ok bool) {
	// need to declare vars early, or else we won't be able to use goto
	//var (
	//	ctx    context.Context
	//	cancel context.CancelFunc
	//)

	chain.mut.Lock()
	block, ok = chain.blocks[hash]
	chain.mut.Unlock()

	if !ok {
		return nil, false
	}

	return block, true

	//	if ok {
	//		goto done
	//	}
	//
	//	ctx, cancel = synchronizer.TimeoutContext(chain.eventLoop.Context(), chain.eventLoop)
	//	chain.pendingFetch[hash] = cancel
	//
	//	chain.mut.Unlock()
	//	chain.logger.Debugf("Attempting to fetch block: %.8s", hash)
	//	block, ok = chain.configuration.Fetch(ctx, hash)
	//	chain.mut.Lock()
	//
	//	delete(chain.pendingFetch, hash)
	//	if !ok {
	//		// check again in case the block arrived while we we fetching
	//		block, ok = chain.blocks[hash]
	//		goto done
	//	}
	//
	//	chain.logger.Debugf("Successfully fetched block: %.8s", hash)
	//
	//	chain.blocks[hash] = block
	//	chain.blockAtHeight[block.View()] = block
	//
	//done:
	//	chain.mut.Unlock()
	//
	//	if !ok {
	//		return nil, false
	//	}
	//
	//	return block, true
}

// Extends checks if the given block extends the branch of the target block.
func (chain *blockChain) Extends(block, target *model.Block) bool {
	current := block
	ok := true
	for ok && current.View() > target.View() {
		current, ok = chain.Get(current.Parent())
	}
	return ok && current.Hash() == target.Hash()
}

//func (chain *blockChain) PruneToHeight(height types.View) (forkedBlocks []*model.Block) {
//	chain.mut.Lock()
//	defer chain.mut.Unlock()
//
//	committedHeight := chain.consensus.CommittedBlock().View()
//	committedViews := make(map[types.View]bool)
//	committedViews[committedHeight] = true
//	for h := committedHeight; h >= chain.pruneHeight; {
//		block, ok := chain.blockAtHeight[h]
//		if !ok {
//			break
//		}
//		parent, ok := chain.blocks[block.Parent()]
//		if !ok || parent.View() < chain.pruneHeight {
//			break
//		}
//		h = parent.View()
//		committedViews[h] = true
//	}
//
//	for h := height; h > chain.pruneHeight; h-- {
//		if !committedViews[h] {
//			block, ok := chain.blockAtHeight[h]
//			if ok {
//				chain.logger.Debugf("PruneToHeight: found forked block: %v", block)
//				forkedBlocks = append(forkedBlocks, block)
//			}
//		}
//		delete(chain.blockAtHeight, h)
//	}
//	chain.pruneHeight = height
//	return forkedBlocks
//}
//
//var _ modules.BlockChain = (*blockChain)(nil)
