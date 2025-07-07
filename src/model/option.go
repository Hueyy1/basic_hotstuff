package model

import (
	"hxy352/src/model/tree"
	"hxy352/src/types"
	"sync"
)

// OptionID is the ID of an option.
type OptionID uint64

// Options stores runtime configuration settings.
type Options struct {
	mut     sync.Mutex
	options []any

	id         types.ID
	privateKey types.PrivateKey

	shouldUseAggQC        bool
	shouldVerifyVotesSync bool

	sharedRandomSeed   int64
	connectionMetadata map[string]string

	tree          tree.Tree
	shouldUseTree bool
}

func (opts *Options) ensureSpace(id OptionID) {
	if int(id) >= len(opts.options) {
		newOpts := make([]any, id+1)
		copy(newOpts, opts.options)
		opts.options = newOpts
	}
}

// SetShouldUseTree sets the ShouldUseTree setting to true.
func (opts *Options) SetShouldUseTree() {
	opts.shouldUseTree = true
}

func (opts *Options) ShouldUseTree() bool {
	return opts.shouldUseTree
}

// Get returns the value associated with the given option ID.
func (opts *Options) Get(id OptionID) any {
	opts.mut.Lock()
	defer opts.mut.Unlock()
	if len(opts.options) <= int(id) {
		return nil
	}
	return opts.options[id]
}

// Set sets the value of the given option ID.
func (opts *Options) Set(id OptionID, value any) {
	opts.mut.Lock()
	defer opts.mut.Unlock()
	opts.ensureSpace(id)
	opts.options[id] = value
}

// ID returns the ID.
func (opts *Options) ID() types.ID {
	return opts.id
}

// PrivateKey returns the private key.
func (opts *Options) PrivateKey() types.PrivateKey {
	return opts.privateKey
}

// ShouldUseAggQC returns true if aggregated quorum certificates should be used.
// This is true for Fast-Hotstuff: https://arxiv.org/abs/2010.11454
func (opts *Options) ShouldUseAggQC() bool {
	return opts.shouldUseAggQC
}

// ShouldVerifyVotesSync returns true if votes should be verified synchronously.
// Enabling this should make the voting machine process votes synchronously.
func (opts *Options) ShouldVerifyVotesSync() bool {
	return opts.shouldVerifyVotesSync
}

// SharedRandomSeed returns a random number that is shared between all replicas.
func (opts *Options) SharedRandomSeed() int64 {
	return opts.sharedRandomSeed
}

// ConnectionMetadata returns the metadata map that is sent when connecting to other replicas.
func (opts *Options) ConnectionMetadata() map[string]string {
	return opts.connectionMetadata
}

// SetShouldUseAggQC sets the ShouldUseAggQC setting to true.
func (opts *Options) SetShouldUseAggQC() {
	opts.shouldUseAggQC = true
}

// SetShouldVerifyVotesSync sets the ShouldVerifyVotesSync setting to true.
func (opts *Options) SetShouldVerifyVotesSync() {
	opts.shouldVerifyVotesSync = true
}

// SetSharedRandomSeed sets the shared random seed.
func (opts *Options) SetSharedRandomSeed(seed int64) {
	opts.sharedRandomSeed = seed
}
