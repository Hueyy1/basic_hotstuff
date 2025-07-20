package crypto

import (
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/types"
)

// CryptoBase provides the basic cryptographic methods needed to create, verify, and combine signatures.
type CryptoBase interface {
	// Sign creates a cryptographic signature of the given message.
	Sign(message []byte) (signature types.QuorumSignature, err error)
	// Combine combines multiple signatures into a single signature.
	Combine(signatures ...types.QuorumSignature) (signature types.QuorumSignature, err error)
	// Verify verifies the given quorum signature against the message.
	Verify(signature types.QuorumSignature, message []byte) bool
	// BatchVerify verifies the given quorum signature against the batch of messages.
	BatchVerify(signature types.QuorumSignature, batch map[types.ID][]byte) bool
}

// Crypto implements the methods required to create and verify signatures and certificates.
// This is a higher level interface that is implemented by the crypto package itself.
type Crypto interface {
	CryptoBase
	// CreatePartialCert signs a single block and returns the partial certificate.
	CreatePartialCert(block *model.Block) (cert types.PartialCert, err error)
	// CreateQuorumCert creates a quorum certificate from a list of partial certificates.
	CreateQuorumCert(block *model.Block, signatures []types.PartialCert) (cert types.QuorumCert, err error)
	// CreateTimeoutCert creates a timeout certificate from a list of timeout messages.
	CreateTimeoutCert(view types.View, timeouts []model.TimeoutMsg) (cert types.TimeoutCert, err error)
	// VerifyPartialCert verifies a single partial certificate.
	VerifyPartialCert(block *model.Block, cert types.PartialCert) bool
	// VerifyQuorumCert verifies a quorum certificate.
	VerifyQuorumCert(block *model.Block, qc types.QuorumCert) bool
	// VerifyTimeoutCert verifies a timeout certificate.
	VerifyTimeoutCert(tc types.TimeoutCert) bool
}

var QuorumSize = 3

type CryptoImpl struct {
	//Bc   model.BlockChain
	Conf *model.ReplicaConf

	CryptoBase
}

// New returns a new implementation of the Crypto interface. It will use the given CryptoBase to create and verify
// signatures.
//func New() Crypto {
//	return &crypto{CryptoBase: impl}
//	//return &crypto{CryptoBase: bls12.New()}
//}

// CreatePartialCert signs a single block and returns the partial certificate.
func (c CryptoImpl) CreatePartialCert(block *model.Block) (cert types.PartialCert, err error) {
	sig, err := c.Sign(block.ToBytes())
	if err != nil {
		return types.PartialCert{}, err
	}
	return types.NewPartialCert(sig, block.Hash()), nil
}

// CreateQuorumCert creates a quorum certificate from a list of partial certificates.
func (c CryptoImpl) CreateQuorumCert(block *model.Block, signatures []types.PartialCert) (cert types.QuorumCert, err error) {
	// genesis QC is always valid.
	if block.Hash() == model.GetGenesis().Hash() {
		return types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()), nil
	}
	sigs := make([]types.QuorumSignature, 0, len(signatures))
	for _, sig := range signatures {
		sigs = append(sigs, sig.Signature())
	}
	sig, err := c.Combine(sigs...)
	if err != nil {
		return types.QuorumCert{}, err
	}
	return types.NewQuorumCert(sig, block.View(), block.Hash()), nil
}

// CreateTimeoutCert creates a timeout certificate from a list of timeout messages.
func (c CryptoImpl) CreateTimeoutCert(view types.View, timeouts []model.TimeoutMsg) (cert types.TimeoutCert, err error) {
	// view 0 is always valid.
	if view == 0 {
		return types.NewTimeoutCert(nil, 0), nil
	}
	sigs := make([]types.QuorumSignature, 0, len(timeouts))
	for _, timeout := range timeouts {
		sigs = append(sigs, timeout.ViewSignature)
	}
	sig, err := c.Combine(sigs...)
	if err != nil {
		return types.TimeoutCert{}, err
	}
	return types.NewTimeoutCert(sig, view), nil
}

// VerifyPartialCert verifies a single partial certificate.
func (c CryptoImpl) VerifyPartialCert(block *model.Block, cert types.PartialCert) bool {
	return c.Verify(cert.Signature(), block.ToBytes())
}

// VerifyQuorumCert verifies a quorum certificate.
func (c CryptoImpl) VerifyQuorumCert(block *model.Block, qc types.QuorumCert) bool {
	// genesis QC is always valid.
	if qc.BlockHash() == model.GetGenesis().Hash() {
		return true
	}

	// TODO: FIX BUG - qcSignature can be nil when a leader is byzantine.
	qcSignature := qc.Signature()
	if qcSignature == nil {
		log.Panicf("quorum certificate has nil signature (view=%d)", qc.View())
	}

	participants := qcSignature.Participants()
	//if participants.Len() < c.configuration.QuorumSize() {
	//	return false
	//}
	if participants.Len() < QuorumSize {
		return false
	}
	return c.Verify(qc.Signature(), block.ToBytes())
}

// VerifyTimeoutCert verifies a timeout certificate.
func (c CryptoImpl) VerifyTimeoutCert(tc types.TimeoutCert) bool {
	// view 0 TC is always valid.
	if tc.View() == 0 {
		return true
	}
	//if tc.Signature().Participants().Len() < c.configuration.QuorumSize() {
	//	return false
	//}
	if tc.Signature().Participants().Len() < QuorumSize {
		return false
	}
	return c.Verify(tc.Signature(), tc.View().ToBytes())
}
