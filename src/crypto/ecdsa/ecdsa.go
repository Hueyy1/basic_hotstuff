// Package ecdsa implements the spec-k256 curve signature.
package ecdsa

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/types"
	"math/big"
)

const (
	// PrivateKeyFileType is the PEM type for a private key.
	PrivateKeyFileType = "ECDSA PRIVATE KEY"
	// PublicKeyFileType is the PEM type for a public key.
	PublicKeyFileType = "ECDSA PUBLIC KEY"
)

var (
	_ types.QuorumSignature = (*Multi[*Signature])(nil)
	_ types.IDSet           = (*Multi[*Signature])(nil)
	_ mSignature            = (*Signature)(nil)
)

var (
	// ErrCombineMultiple is used when Combine is called with less than two signatures.
	ErrCombineMultiple = errors.New("must have at least two signatures")

	// ErrCombineOverlap is used when Combine is called with signatures that have overlapping participation.
	ErrCombineOverlap = errors.New("overlapping signatures")
)

// Signature is an ECDSA signature.
type Signature struct {
	r, s   *big.Int
	signer types.ID
}

// RestoreSignature restores an existing signature.
// It should not be used to create new signatures, use Sign instead.
func RestoreSignature(r, s *big.Int, signer types.ID) *Signature {
	return &Signature{r, s, signer}
}

// Signer returns the ID of the replica that generated the signature.
func (sig Signature) Signer() types.ID {
	return sig.signer
}

// R returns the r value of the signature.
func (sig Signature) R() *big.Int {
	return sig.r
}

// S returns the s value of the signature.
func (sig Signature) S() *big.Int {
	return sig.s
}

// ToBytes returns a raw byte string representation of the signature.
func (sig Signature) ToBytes() []byte {
	var b []byte
	b = append(b, sig.r.Bytes()...)
	b = append(b, sig.s.Bytes()...)
	return b
}

type EcdsaBase struct {
	//configuration modules.Configuration
	//logger        logging.Logger
	//opts          *modules.Options

	conf  *model.ReplicaConf
	gConf *model.Config
}

// New returns a new instance of the ECDSA CryptoBase implementation.
func New(conf *model.ReplicaConf, gConf *model.Config) *EcdsaBase {
	return &EcdsaBase{gConf: gConf, conf: conf}
}

func (ec *EcdsaBase) privateKey() *ecdsa.PrivateKey {
	return ec.conf.PriKey.(*ecdsa.PrivateKey)
}

// Sign creates a cryptographic signature of the given message.
func (ec *EcdsaBase) Sign(message []byte) (signature types.QuorumSignature, err error) {
	hash := sha256.Sum256(message)
	r, s, err := ecdsa.Sign(rand.Reader, ec.privateKey(), hash[:])
	if err != nil {
		return nil, fmt.Errorf("ecdsa: sign failed: %w", err)
	}
	return Multi[*Signature]{ec.conf.Id: &Signature{
		r:      r,
		s:      s,
		signer: ec.conf.Id,
	}}, nil
}

// Combine combines multiple signatures into a single signature.
func (ec *EcdsaBase) Combine(signatures ...types.QuorumSignature) (types.QuorumSignature, error) {
	if len(signatures) < 2 {
		return nil, ErrCombineMultiple
	}

	ts := make(Multi[*Signature])
	for _, sig1 := range signatures {
		if sig2, ok := sig1.(Multi[*Signature]); ok {
			for id, s := range sig2 {
				if _, duplicate := ts[id]; duplicate {
					return nil, ErrCombineOverlap
				}
				ts[id] = s
			}
		} else {
			log.Panicf("cannot combine signature of incompatible type %T (expected %T)", sig1, sig2)
		}
	}
	return ts, nil
}

// Verify verifies the given quorum signature against the message.
func (ec *EcdsaBase) Verify(signature types.QuorumSignature, message []byte) bool {
	s, ok := signature.(Multi[*Signature])
	if !ok {
		log.Panicf("cannot verify signature of incompatible type %T (expected %T)", signature, s)
	}
	n := signature.Participants().Len()
	if n == 0 {
		return false
	}

	results := make(chan bool, n)
	hash := sha256.Sum256(message)
	for _, sig := range s {
		go func(sig *Signature, hash types.Hash) {
			results <- ec.verifySingle(sig, hash)
		}(sig, hash)
	}
	valid := true
	for range s {
		if !<-results {
			valid = false
		}
	}
	return valid
}

func (ec *EcdsaBase) verifySingle(sig *Signature, hash types.Hash) bool {
	r := ec.gConf.ReplicaConf[int(sig.Signer())]
	pk := r.PubKey.(*ecdsa.PublicKey)
	return ecdsa.Verify(pk, hash[:], sig.R(), sig.S())
}
