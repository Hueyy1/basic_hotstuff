package basichotstuffpb

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"hxy352/src/crypto"
	"hxy352/src/crypto/ecdsa"
	"hxy352/src/model"
	"hxy352/src/types"
	"math/big"
)

// QuorumSignatureFromProto converts a protocol buffers message to a threshold signature.
func QuorumSignatureFromProto(sig *QuorumSignature) types.QuorumSignature {
	if signature := sig.GetECDSASigs(); signature != nil {
		sigs := make([]*ecdsa.Signature, len(signature.GetSigs()))
		for i, sig := range signature.GetSigs() {
			r := new(big.Int)
			r.SetBytes(sig.GetR())
			s := new(big.Int)
			s.SetBytes(sig.GetS())
			sigs[i] = ecdsa.RestoreSignature(r, s, types.ID(sig.GetSigner()))
		}
		return crypto.Restore(sigs)
	}
	return nil
}

// QuorumCertFromProto converts a basichotstuffpb.QuorumCert to a QuorumCert.
func QuorumCertFromProto(qc *QuorumCert) types.QuorumCert {
	var h types.Hash
	copy(h[:], qc.GetHash())
	return types.NewQuorumCert(QuorumSignatureFromProto(qc.GetSig()), types.View(qc.GetView()), h)
}

// QuorumSignatureToProto converts a threshold signature to a protocol buffers message.
func QuorumSignatureToProto(sig types.QuorumSignature) *QuorumSignature {
	signature := &QuorumSignature{}

	if sig == nil {
		return signature
	}

	sigs := make([]*ECDSASignature, 0, sig.Participants().Len())
	for _, s := range sig.(crypto.Multi[*ecdsa.Signature]) {
		sigs = append(sigs, &ECDSASignature{
			Signer: uint32(s.Signer()),
			R:      s.R().Bytes(),
			S:      s.S().Bytes(),
		})
	}
	signature.Sig = &QuorumSignature_ECDSASigs{ECDSASigs: &ECDSAMultiSignature{
		Sigs: sigs,
	}}

	return signature
}

// QuorumCertToProto converts a consensus.QuorumCert to a hotstuffpb.QuorumCert.
func QuorumCertToProto(qc types.QuorumCert) *QuorumCert {
	hash := qc.BlockHash()
	return &QuorumCert{
		Sig:  QuorumSignatureToProto(qc.Signature()),
		Hash: hash[:],
		View: uint64(qc.View()),
	}
}

// BlockToProto converts a consensus.Block to a hotstuffpb.Block.
func BlockToProto(block *model.Block) *Block {
	parentHash := block.Parent()
	h := block.Hash()
	return &Block{
		Parent:    parentHash[:],
		Hash:      h[:],
		Command:   []byte(block.Command()),
		QC:        QuorumCertToProto(block.QuorumCert()),
		View:      uint64(block.View()),
		Proposer:  uint32(block.Proposer()),
		Timestamp: timestamppb.New(block.Timestamp()),
	}
}

// BlockFromProto converts a hotstuffpb.Block to a consensus.Block.
func BlockFromProto(block *Block) *model.Block {
	var p types.Hash
	copy(p[:], block.GetParent())

	b := model.NewBlock(
		p,
		QuorumCertFromProto(block.GetQC()),
		types.Command(block.GetCommand()),
		types.View(block.GetView()),
		types.ID(block.GetProposer()),
	)
	b.SetTimestamp(block.Timestamp.AsTime())
	return b
}

// PartialCertToProto converts a consensus.PartialCert to a hotstuffpb.PartialCert.
func PartialCertToProto(cert types.PartialCert) *PartialCert {
	hash := cert.BlockHash()
	return &PartialCert{
		Sig:  QuorumSignatureToProto(cert.Signature()),
		Hash: hash[:],
	}
}

// PartialCertFromProto converts a hotstuffpb.PartialCert to an ecdsa.PartialCert.
func PartialCertFromProto(cert *PartialCert) types.PartialCert {
	var h types.Hash
	copy(h[:], cert.GetHash())
	return types.NewPartialCert(QuorumSignatureFromProto(cert.GetSig()), h)
}

// TimeoutCertFromProto converts a timeout certificate from the protobuf type to the hotstuff type.
func TimeoutCertFromProto(m *TimeoutCert) types.TimeoutCert {
	return types.NewTimeoutCert(QuorumSignatureFromProto(m.GetSig()), types.View(m.GetView()))
}

// TimeoutCertToProto converts a timeout certificate from the hotstuff type to the protobuf type.
func TimeoutCertToProto(timeoutCert types.TimeoutCert) *TimeoutCert {
	return &TimeoutCert{
		View: uint64(timeoutCert.View()),
		Sig:  QuorumSignatureToProto(timeoutCert.Signature()),
	}
}
