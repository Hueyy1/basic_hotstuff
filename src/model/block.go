package model

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hxy352/src/types"
	"time"
)

// Block contains a proposed "command", metadata for the protocol, and a link to the "parent" block.
type Block struct {
	// keep a copy of the hash to avoid hashing multiple times
	hash     types.Hash
	parent   types.Hash
	proposer types.ID
	cmd      types.Command
	cert     types.QuorumCert
	view     types.View
	ts       time.Time
}

// NewBlock creates a new Block
func NewBlock(parent types.Hash, cert types.QuorumCert, cmd types.Command, view types.View, proposer types.ID) *Block {
	b := &Block{
		parent:   parent,
		cert:     cert,
		cmd:      cmd,
		view:     view,
		proposer: proposer,
		ts:       time.Now(),
	}
	// cache the hash immediately because it is too racy to do it in Hash()
	b.hash = sha256.Sum256(b.ToBytes())
	return b
}

func (b *Block) SetTimestamp(ts time.Time) {
	b.ts = ts
	// recalculate the hash since the timestamp is part of the block
	b.hash = sha256.Sum256(b.ToBytes())
}

func (b *Block) String() string {
	return fmt.Sprintf(
		"Block{ hash: %.6s parent: %.6s, proposer: %d, view: %d , cert: %v }",
		b.Hash().String(),
		b.parent.String(),
		b.proposer,
		b.view,
		b.cert,
	)
}

// Hash returns the hash of the Block
func (b *Block) Hash() types.Hash {
	return b.hash
}

// Proposer returns the id of the replica who proposed the block.
func (b *Block) Proposer() types.ID {
	return b.proposer
}

// Parent returns the hash of the parent Block
func (b *Block) Parent() types.Hash {
	return b.parent
}

// Command returns the command
func (b *Block) Command() types.Command {
	return b.cmd
}

// QuorumCert returns the quorum certificate in the block
func (b *Block) QuorumCert() types.QuorumCert {
	return b.cert
}

// View returns the view in which the Block was proposed
func (b *Block) View() types.View {
	return b.view
}

// Timestamp returns the timestamp of the block
func (b *Block) Timestamp() time.Time {
	return b.ts
}

// ToBytes returns the raw byte form of the Block, to be used for hashing, etc.
func (b *Block) ToBytes() []byte {
	buf := b.parent[:]
	var proposerBuf [4]byte
	binary.LittleEndian.PutUint32(proposerBuf[:], uint32(b.proposer))
	buf = append(buf, proposerBuf[:]...)
	var viewBuf [8]byte
	binary.LittleEndian.PutUint64(viewBuf[:], uint64(b.view))
	buf = append(buf, viewBuf[:]...)
	buf = append(buf, []byte(b.cmd)...)
	buf = append(buf, b.cert.ToBytes()...)
	var tsBuf [8]byte
	binary.LittleEndian.PutUint64(tsBuf[:], uint64(b.ts.UnixNano()))
	buf = append(buf, tsBuf[:]...)
	return buf
}

// genesisBlock is initialized at package initialization time.
var genesisBlock = func() *Block {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	b := NewBlock(types.Hash{}, types.QuorumCert{}, "", 0, 0)
	b.SetTimestamp(ts)
	return b
}()

// GetGenesis returns the genesis block.
func GetGenesis() *Block {
	return genesisBlock
}
