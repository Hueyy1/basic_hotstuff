package testutil

import (
	"hxy352/src/crypto"
	"hxy352/src/model"
	"hxy352/src/types"
	"testing"
)

// CreatePC creates a partial certificate using the given signer.
func CreatePC(t *testing.T, block *model.Block, signer crypto.Crypto) types.PartialCert {
	t.Helper()
	pc, err := signer.CreatePartialCert(block)
	if err != nil {
		t.Fatalf("Failed to create partial certificate: %v", err)
	}
	return pc
}

// CreatePCs creates one partial certificate using each of the given signers.
func CreatePCs(t *testing.T, block *model.Block, signers []crypto.Crypto) []types.PartialCert {
	t.Helper()
	pcs := make([]types.PartialCert, 0, len(signers))
	for _, signer := range signers {
		pcs = append(pcs, CreatePC(t, block, signer))
	}
	return pcs
}
