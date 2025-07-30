package ecdsa

import (
	"hxy352/src/types"
	"slices"
)

// Signature is the individual component in MultiSignature
type mSignature interface {
	Signer() types.ID
	ToBytes() []byte
}

// Multi is a set of (partial) signatures.
type Multi[T mSignature] map[types.ID]T

// Restore should only be used to restore an existing threshold signature from a set of signatures.
func Restore[T mSignature](signatures []T) Multi[T] {
	sig := make(Multi[T], len(signatures))
	for _, s := range signatures {
		sig[s.Signer()] = s
	}
	return sig
}

// ToBytes returns the object as bytes.
func (sig Multi[T]) ToBytes() []byte {
	var b []byte
	// sort by ID to make it deterministic
	order := make([]types.ID, 0, len(sig))
	for _, signature := range sig {
		order = append(order, signature.Signer())
	}
	slices.Sort(order)
	for _, id := range order {
		b = append(b, sig[id].ToBytes()...)
	}
	return b
}

// Participants returns the IDs of replicas who participated in the threshold signature.
func (sig Multi[T]) Participants() types.IDSet {
	return sig
}

// Add adds an ID to the set.
func (sig Multi[T]) Add(_ types.ID) {
	panic("not implemented")
}

// Contains returns true if the set contains the ID.
func (sig Multi[T]) Contains(id types.ID) bool {
	_, ok := sig[id]
	return ok
}

// ForEach calls f for each ID in the set.
func (sig Multi[T]) ForEach(f func(types.ID)) {
	for id := range sig {
		f(id)
	}
}

// RangeWhile calls f for each ID in the set until f returns false.
func (sig Multi[T]) RangeWhile(f func(types.ID) bool) {
	for id := range sig {
		if !f(id) {
			break
		}
	}
}

// Len returns the number of entries in the set.
func (sig Multi[T]) Len() int {
	return len(sig)
}

func (sig Multi[T]) String() string {
	return types.IDSetToString(sig)
}
