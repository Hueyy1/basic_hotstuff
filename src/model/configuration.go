package model

import (
	"context"
	"hxy352/src/types"
)

// Configuration holds information about the current configuration of replicas that participate in the protocol,
// It provides methods to send messages to the other replicas.
type Configuration interface {
	// Replicas returns all of the replicas in the configuration.
	Replicas() map[types.ID]Replica
	// Replica returns a replica if present in the configuration.
	Replica(types.ID) (replica Replica, ok bool)
	// Len returns the number of replicas in the configuration.
	Len() int
	// QuorumSize returns the size of a quorum.
	QuorumSize() int
	// Propose sends the block to all replicas in the configuration.
	Propose(proposal ProposeMsg)
	// Timeout sends the timeout message to all replicas.
	Timeout(msg TimeoutMsg)
	// Fetch requests a block from all the replicas in the configuration.
	Fetch(ctx context.Context, hash types.Hash) (block *Block, ok bool)
	// SubConfig returns a subconfiguration containing the replicas specified in the ids slice.
	SubConfig(ids []types.ID) (sub Configuration, err error)
}
