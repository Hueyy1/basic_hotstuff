package crypto

import (
	"fmt"
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
	CreateTimeoutCert(view types.View, sigs []types.QuorumSignature) (cert types.TimeoutCert, err error)
	// VerifyPartialCert verifies a single partial certificate.
	VerifyPartialCert(block *model.Block, cert types.PartialCert) bool
	// VerifyQuorumCert verifies a quorum certificate.
	VerifyQuorumCert(block *model.Block, qc types.QuorumCert) bool
	// VerifyTimeoutCert verifies a timeout certificate.
	VerifyTimeoutCert(tc types.TimeoutCert) bool

	CreateTimeoutVoteCert(tv types.TimeoutVote) (cert types.QuorumSignature, err error)
	VerifyTimeoutVoteCert(tv types.TimeoutVote, sig types.QuorumSignature) bool
	CreateTimeoutQuorumCert(tv types.TimeoutVote, sigs []types.QuorumSignature) (cert types.QuorumCert, err error)
	VerifyTimeoutQuorumCert(tv types.TimeoutVote, qc types.QuorumCert) bool
}

// todo: quick show
func hotstuffQuorum(n int) (maxFaulty int, minCorrect int, err error) {
	if n < 4 {
		return 0, 0, fmt.Errorf("n must be >= 4")
	}
	maxFaulty = (n - 1) / 3
	minCorrect = n - maxFaulty
	return maxFaulty, minCorrect, nil
}

// QuorumSize 2f + 1, default = 3
var QuorumSize = 3

// FaultSize f, default = 1
var FaultSize = 1

// real fault node, default = 0
// gConf.FaultNumber
var IsFaultNode = false

type CryptoImpl struct {
	//Bc   model.BlockChain
	GConf *model.Config
	Conf  *model.ReplicaConf

	CryptoBase
}

func (c CryptoImpl) Set_Normal_And_Fault_Size(f int) {

	if f == 0 {
		return
	}

	FaultSize = f
	QuorumSize = 2*f + 1

	c.Set_Fault_Nodes()

	return
}

// Set_Fault_Nodes return nodes which should perform fault
func (c CryptoImpl) Set_Fault_Nodes() []types.ID {
	if c.GConf.FaultNumber == 0 {
		return []types.ID{}
	}

	// F    1/2/3/ 4/ 5
	// node 3/6/9/12/15
	res := make([]types.ID, 0, c.GConf.FaultNumber)
	for i := 0; i < c.GConf.FaultNumber; i++ {
		tmp := types.ID(i*3 + 3)
		res = append(res, tmp)

		if tmp == c.Conf.Id {
			IsFaultNode = true
		}
	}
	return res
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
func (c CryptoImpl) CreateTimeoutCert(view types.View, sigs []types.QuorumSignature) (cert types.TimeoutCert, err error) {
	// view 0 is always valid.
	if view == 0 {
		return types.NewTimeoutCert(nil, 0), nil
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
	if tc.Signature().Participants().Len() < FaultSize+1 {
		return false
	}
	return c.Verify(tc.Signature(), tc.View().ToBytes())
}

func (c CryptoImpl) CreateTimeoutVoteCert(tv types.TimeoutVote) (cert types.QuorumSignature, err error) {
	return c.Sign(tv.ToBytes())
}

func (c CryptoImpl) VerifyTimeoutVoteCert(tv types.TimeoutVote, sig types.QuorumSignature) bool {
	return c.Verify(sig, tv.ToBytes())
}

func (c CryptoImpl) CreateTimeoutQuorumCert(tv types.TimeoutVote, sigs []types.QuorumSignature) (cert types.QuorumCert, err error) {
	sig, err := c.Combine(sigs...)
	if err != nil {
		return types.QuorumCert{}, err
	}
	return types.NewQuorumCert(sig, tv.View, tv.TC.Hash()), nil
}

func (c CryptoImpl) VerifyTimeoutQuorumCert(tv types.TimeoutVote, qc types.QuorumCert) bool {
	qcSignature := qc.Signature()
	if qcSignature == nil {
		log.Panicf("quorum certificate has nil signature (view=%d)", qc.View())
	}

	participants := qcSignature.Participants()
	if participants.Len() < QuorumSize {
		return false
	}
	return c.Verify(qc.Signature(), tv.ToBytes())
}
