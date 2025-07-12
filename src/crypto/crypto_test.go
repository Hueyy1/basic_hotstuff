package crypto_test

import (
	"go.uber.org/mock/gomock"
	"hxy352/src/crypto"
	"hxy352/src/crypto/bls12"
	"hxy352/src/model"
	"hxy352/src/testutil"
	"hxy352/src/types"
	"testing"
)

func TestCreatePartialCert(t *testing.T) {
	run := func(t *testing.T, setup setupFunc) {
		ctrl := gomock.NewController(t)

		td := setup(t, ctrl, 1)

		partialCert, err := td.signers[0].CreatePartialCert(td.block)
		if err != nil {
			t.Fatalf("Failed to create partial certificate: %v", err)
		}

		if partialCert.BlockHash() != td.block.Hash() {
			t.Error("Partial certificate hash does not match block hash!")
		}

		if signerID := partialCert.Signer(); signerID != types.ID(1) {
			t.Errorf("Wrong ID for signer in partial certificate: got: %d, want: %d", signerID, types.ID(1))
		}
	}
	runAll(t, run)
}

func TestVerifyPartialCert(t *testing.T) {
	run := func(t *testing.T, setup setupFunc) {
		ctrl := gomock.NewController(t)

		td := setup(t, ctrl, 1)
		partialCert := testutil.CreatePC(t, td.block, td.signers[0])

		if !td.verifiers[0].VerifyPartialCert(partialCert) {
			t.Error("Partial Certificate was not verified.")
		}
	}
	runAll(t, run)
}

func TestCreateQuorumCert(t *testing.T) {
	run := func(t *testing.T, setup setupFunc) {
		ctrl := gomock.NewController(t)

		td := setup(t, ctrl, 4)

		pcs := testutil.CreatePCs(t, td.block, td.signers)

		qc, err := td.signers[0].CreateQuorumCert(td.block, pcs)
		if err != nil {
			t.Fatalf("Failed to create QC: %v", err)
		}

		if qc.BlockHash() != td.block.Hash() {
			t.Error("Quorum certificate hash does not match block hash!")
		}
	}
	runAll(t, run)
}

func TestVerifyQuorumCert(t *testing.T) {
	run := func(t *testing.T, setup setupFunc) {
		ctrl := gomock.NewController(t)

		td := setup(t, ctrl, 4)

		qc := testutil.CreateQC(t, td.block, td.signers)

		for i, verifier := range td.verifiers {
			if !verifier.VerifyQuorumCert(qc) {
				t.Errorf("verifier %d failed to verify QC!", i+1)
			}
		}
	}
	runAll(t, run)
}

func runAll(t *testing.T, run func(*testing.T, setupFunc)) {
	t.Helper()
	t.Run("BLS12-381", func(t *testing.T) { run(t, setup(NewBase(bls12.New), testutil.GenerateBLS12Key)) })
}

func createBlock(t *testing.T, signer modules.Crypto) *hotstuff.Block {
	t.Helper()

	qc, err := signer.CreateQuorumCert(hotstuff.GetGenesis(), []hotstuff.PartialCert{})
	if err != nil {
		t.Errorf("Could not create empty QC for genesis: %v", err)
	}

	b := hotstuff.NewBlock(hotstuff.GetGenesis().Hash(), qc, "foo", 42, 1)
	return b
}

type keyFunc func(t *testing.T) hotstuff.PrivateKey
type setupFunc func(*testing.T, *gomock.Controller, int) testData

func setup(newFunc func() modules.Crypto, keyFunc keyFunc) setupFunc {
	return func(t *testing.T, ctrl *gomock.Controller, n int) testData {
		return newTestData(t, ctrl, n, newFunc, keyFunc)
	}
}

func NewCache(impl func() modules.CryptoBase) func() modules.Crypto {
	return func() modules.Crypto {
		return crypto.NewCache(impl(), 10)
	}
}

func NewBase(impl func() modules.CryptoBase) func() modules.Crypto {
	return func() modules.Crypto {
		return crypto.New(impl())
	}
}

type testData struct {
	signers   []crypto.Crypto
	verifiers []crypto.Crypto
	block     *model.Block
}

func newTestData(t *testing.T, ctrl *gomock.Controller, n int, newFunc func() modules.Crypto, keyFunc keyFunc) testData {
	t.Helper()

	bl := testutil.CreateBuilders(t, ctrl, n, testutil.GenerateKeys(t, n, keyFunc)...)
	for _, builder := range bl {
		signer := newFunc()
		builder.Add(signer)
	}
	hl := bl.Build()

	var signer modules.Crypto
	hl[0].Get(&signer)

	block := createBlock(t, signer)

	for _, mods := range hl {
		var blockChain modules.BlockChain
		mods.Get(&blockChain)

		blockChain.Store(block)
	}

	return testData{
		signers:   hl.Signers(),
		verifiers: hl.Verifiers(),
		block:     block,
	}
}
