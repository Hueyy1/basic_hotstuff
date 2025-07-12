package basichotstuffpb

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"hxy352/src/crypto/bitfield"
	"hxy352/src/crypto/bls12"
	"hxy352/src/model"
	"hxy352/src/types"
)

// QuorumSignatureFromProto converts a protocol buffers message to a threshold signature.
func QuorumSignatureFromProto(sig *QuorumSignature) types.QuorumSignature {
	if signature := sig.GetBLS12Sig(); signature != nil {
		aggSig, err := bls12.RestoreAggregateSignature(signature.GetSig(), bitfield.BitfieldFromBytes(signature.GetParticipants()))
		if err != nil {
			return nil
		}
		return aggSig
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
	switch ms := sig.(type) {

	case *bls12.AggregateSignature:
		signature.Sig = &QuorumSignature_BLS12Sig{BLS12Sig: &BLS12AggregateSignature{
			Sig:          ms.ToBytes(),
			Participants: ms.Bitfield().Bytes(),
		}}
	}
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
