package model

import (
	"crypto/ecdsa"
	"crypto/x509"
	"hxy352/src/types"
)

// Replica represents a remote replica participating in the consensus protocol.
// The methods Vote, NewView, and Deliver must send the respective arguments to the remote replica.
type Replica interface {
	// ID returns the replica's id.
	ID() types.ID
	// PublicKey returns the replica's public key.
	PublicKey() types.PublicKey
	// Vote sends the partial certificate to the other replica.
	Vote(cert types.PartialCert)
	// NewView sends the quorum certificate to the other replica.
	NewView(types.SyncInfo)
	// Metadata returns the connection metadata sent by this replica.
	Metadata() map[string]string
}

type ReplicaConf struct {
	Id                   types.ID
	Conf                 ReplicaConfig
	CaKey                *ecdsa.PrivateKey
	Ca                   *x509.Certificate
	CertificateAuthority []byte
	Crypto               string `default:"bls12"`
	PrivateKey           []byte
	PublicKey            []byte
	Certificate          []byte
	CertificateKey       []byte
	UseTLS               bool
	//RootCAs              *x509.CertPool
	PrivKey types.PrivateKey
	//TlsCertificate       tls.Certificate
}
