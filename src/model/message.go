package model

import (
	"bytes"
	"fmt"
	"hxy352/src/types"
)

// OpCode 定义 opcode 类型和常量
type OpCode uint8

const (
	OpCodePropose OpCode = 0
)

// ProposeMsg is broadcast when a leader makes a proposal.
type ProposeMsg struct {
	ID    types.ID // The ID of the replica who sent the message.
	Block *Block   // The block that is proposed.
}

func (p ProposeMsg) String() string {
	return fmt.Sprintf("ID %d, %s", p.ID, p.Block)
}

// TimeoutMsg is broadcast whenever a replica has a local timeout.
type TimeoutMsg struct {
	ID            types.ID              // The ID of the replica who sent the message.
	View          types.View            // The view that the replica wants to enter.
	ViewSignature types.QuorumSignature // A signature of the view
	MsgSignature  types.QuorumSignature // A signature of the view, QC.BlockHash, and the replica ID
	SyncInfo      types.SyncInfo        // The highest QC/TC known to the sender.
}

// ToBytes returns a byte form of the timeout message.
func (timeout TimeoutMsg) ToBytes() []byte {
	var b bytes.Buffer
	_, _ = b.Write(timeout.ID.ToBytes())
	_, _ = b.Write(timeout.View.ToBytes())
	if qc, ok := timeout.SyncInfo.QC(); ok {
		_, _ = b.Write(qc.ToBytes())
	}
	return b.Bytes()
}

func (timeout TimeoutMsg) String() string {
	return fmt.Sprintf("ID: %d, View: %d, SyncInfo: %v", timeout.ID, timeout.View, timeout.SyncInfo)
}
