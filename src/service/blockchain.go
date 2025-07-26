package service

import (
	"hxy352/src/log"
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

	chain.mut.Lock()
	block, ok = chain.blocks[hash]
	chain.mut.Unlock()

	if !ok {
		return nil, false
	}

	return block, true

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

func (chain *blockChain) Clean(block *model.Block) {

	parent, ok := chain.Get(block.Parent())
	if !ok {
		log.Warnf("BlockChain Clean: Get parent block err")
		return
	}

	grandParent, ok := chain.Get(parent.Parent())
	if !ok {
		log.Warnf("BlockChain Clean: Get grandParent block err")
		return
	}

	target, ok := chain.Get(grandParent.Parent())
	if !ok {
		log.Warnf("BlockChain Clean: Get target clean block err")
		return
	}

	chain.mut.Lock()
	defer chain.mut.Unlock()
	delete(chain.blocks, target.Hash())
	delete(chain.blockAtHeight, target.View())

	log.Debugf("BlockChain Clean result: blocks len: %d; blockAtHeight len: %d", len(chain.blocks), len(chain.blockAtHeight))
}
