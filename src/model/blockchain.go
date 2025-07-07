package model

import "hxy352/src/types"

// BlockChain is a datastructure that stores a chain of blocks.
// It is not required that a block is stored forever,
// but a block must be stored until at least one of its children have been committed.
type BlockChain interface {
	// Store stores a block in the blockchain.
	Store(*Block)

	// Get retrieves a block given its hash, attempting to fetching it from other replicas if necessary.
	Get(types.Hash) (*Block, bool)

	// LocalGet retrieves a block given its hash, without fetching it from other replicas.
	LocalGet(types.Hash) (*Block, bool)

	// Extends checks if the given block extends the branch of the target hash.
	Extends(block, target *Block) bool
	//
	//// Prunes blocks from the in-memory tree up to the specified height.
	//// Returns a set of forked blocks (blocks that were on a different branch, and thus not committed).
	//PruneToHeight(height types.View) (forkedBlocks []*Block)
}
