package bls12

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	bls12 "github.com/kilic/bls12-381"
	"hxy352/src/crypto"
	"hxy352/src/crypto/bitfield"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/types"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// PrivateKeyFileType is the PEM types for a private key.
	PrivateKeyFileType = "BLS12-381 PRIVATE KEY"

	// PublicKeyFileType is the PEM types for a public key.
	PublicKeyFileType = "BLS12-381 PUBLIC KEY"

	popMetadataKey = "bls12-pop-bin"
)

var (
	domain    = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")
	domainPOP = []byte("BLS_POP_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

	// the order r of G1
	curveOrder, _ = new(big.Int).SetString("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 16)
)

// PublicKey represents a BLS12 public key.
type PublicKey struct {
	p *bls12.PointG1
}

// ToBytes marshals the public key to a byte slice.
func (pub PublicKey) ToBytes() []byte {
	return bls12.NewG1().ToCompressed(pub.p)
}

// FromBytes unmarshals the public key from a byte slice.
func (pub *PublicKey) FromBytes(b []byte) (err error) {
	pub.p, err = bls12.NewG1().FromCompressed(b)
	if err != nil {
		return fmt.Errorf("bls12: failed to decompress public key: %w", err)
	}
	return nil
}

// PrivateKey is a bls12-381 private key.
type PrivateKey struct {
	p *big.Int
}

// ToBytes marshals the private key to a byte slice.
func (pri PrivateKey) ToBytes() []byte {
	return pri.p.Bytes()
}

// FromBytes unmarshals the private key from a byte slice.
func (pri *PrivateKey) FromBytes(b []byte) {
	pri.p = new(big.Int)
	pri.p.SetBytes(b)
}

// Public returns the public key associated with this private key.
func (pri *PrivateKey) Public() types.PublicKey {
	p := &bls12.PointG1{}
	// The public key is the secret key multiplied by the generator G1
	return &PublicKey{p: bls12.NewG1().MulScalarBig(p, &bls12.G1One, pri.p)}
}

// GeneratePrivateKey generates a new private key.
func GeneratePrivateKey() (*PrivateKey, error) {
	// the private key is uniformly random integer such that 0 <= pk < r
	pk, err := rand.Int(rand.Reader, curveOrder)
	if err != nil {
		return nil, fmt.Errorf("bls12: failed to generate private key: %w", err)
	}
	return &PrivateKey{
		p: pk,
	}, nil
}

// AggregateSignature is a bls12-381 aggregate signature. The participants field contains the IDs of the replicas that
// participated in signature creation. This allows us to build an aggregated public key to verify the signature.
type AggregateSignature struct {
	sig          bls12.PointG2
	participants bitfield.Bitfield // The ids of the replicas who submitted signatures.
}

// RestoreAggregateSignature restores an existing aggregate signature. It should not be used to create new aggregate
// signatures. Use CreateThresholdSignature instead.
func RestoreAggregateSignature(sig []byte, participants bitfield.Bitfield) (s *AggregateSignature, err error) {
	p, err := bls12.NewG2().FromCompressed(sig)
	if err != nil {
		return nil, fmt.Errorf("bls12: failed to restore aggregate signature: %w", err)
	}
	return &AggregateSignature{
		sig:          *p,
		participants: participants,
	}, nil
}

// ToBytes returns a byte representation of the aggregate signature.
func (agg *AggregateSignature) ToBytes() []byte {
	if agg == nil {
		return nil
	}
	b := bls12.NewG2().ToCompressed(&agg.sig)
	return b
}

// Participants returns the IDs of replicas who participated in the threshold signature.
func (agg AggregateSignature) Participants() types.IDSet {
	return &agg.participants
}

// Bitfield returns the bitmask.
func (agg AggregateSignature) Bitfield() bitfield.Bitfield {
	return agg.participants
}

func firstParticipant(participants types.IDSet) types.ID {
	id := types.ID(0)
	participants.RangeWhile(func(i types.ID) bool {
		id = i
		return false
	})
	return id
}

type bls12Base struct {
	priKey  *PrivateKey
	replica model.Replica

	//opts *model.Options

	mut sync.RWMutex
	// popCache caches the proof-of-possession results of popVerify for each public key.
	popCache map[string]bool

	conf  *model.ReplicaConf
	gConf *model.Config
}

// New returns a new instance of the BLS12 CryptoBase implementation.
func New(conf *model.ReplicaConf) crypto.CryptoBase {
	var pk PrivateKey
	pk.FromBytes(conf.PrivateKey)

	return &bls12Base{
		popCache: make(map[string]bool),
		conf:     conf,
		//priKey: &PrivateKey{},
		priKey: &pk,
	}
}

func (bls *bls12Base) privateKey() *PrivateKey {
	//log.Infof("opt: %+v", bls.opts)
	//return bls.opts.PrivateKey().(*PrivateKey)
	return bls.priKey
}

func (bls *bls12Base) publicKey(id types.ID) (pubKey *PublicKey, ok bool) {

	//if replica, ok := bls.configuration.Replica(id); ok {
	//	if replica.ID() != bls.opts.ID() && !bls.checkPop(replica) {
	//		bls.logger.Warnf("Invalid POP for replica %d", id)
	//		return nil, false
	//	}
	//	if pubKey, ok = replica.PublicKey().(*PublicKey); ok {
	//		return pubKey, true
	//	}
	//	bls.logger.Errorf("Unsupported public key types: %T", replica.PublicKey())
	//}
	return nil, false
}

func (bls *bls12Base) subgroupCheck(point *bls12.PointG2) bool {
	var p bls12.PointG2
	g2 := bls12.NewG2()
	g2.MulScalarBig(&p, point, curveOrder)
	return g2.IsZero(&p)
}

func (bls *bls12Base) coreSign(message []byte, domainTag []byte) (*bls12.PointG2, error) {
	pk := bls.privateKey()
	g2 := bls12.NewG2()
	point, err := g2.HashToCurve(message, domainTag)
	if err != nil {
		return nil, err
	}
	// multiply the point by the secret key, storing the result in the same point variable
	g2.MulScalarBig(point, point, pk.p)
	return point, nil
}

func (bls *bls12Base) coreVerify(pubKey *PublicKey, message []byte, signature *bls12.PointG2, domainTag []byte) bool {
	if !bls.subgroupCheck(signature) {
		return false
	}
	g2 := bls12.NewG2()
	messagePoint, err := g2.HashToCurve(message, domainTag)
	if err != nil {
		return false
	}
	engine := bls12.NewEngine()
	engine.AddPairInv(&bls12.G1One, signature)
	engine.AddPair(pubKey.p, messagePoint)
	return engine.Result().IsOne()
}

func (bls *bls12Base) popProve() *bls12.PointG2 {
	pubKey := bls.privateKey().Public().(*PublicKey)
	proof, err := bls.coreSign(pubKey.ToBytes(), domainPOP)
	if err != nil {
		log.Panicf("Failed to generate proof-of-possession: %v", err)
	}
	return proof
}

func (bls *bls12Base) popVerify(pubKey *PublicKey, proof *bls12.PointG2) bool {
	return bls.coreVerify(pubKey, pubKey.ToBytes(), proof, domainPOP)
}

func (bls *bls12Base) checkPop(replica model.Replica) (valid bool) {
	defer func() {
		if !valid {
			log.Warnf("Invalid proof-of-possession for replica %d", replica.ID())
		}
	}()

	popBytes, ok := replica.Metadata()[popMetadataKey]
	if !ok {
		log.Warnf("Missing proof-of-possession for replica: %d", replica.ID())
		return false
	}

	var key strings.Builder
	key.WriteString(popBytes)
	_, _ = key.Write(replica.PublicKey().(*PublicKey).ToBytes())

	bls.mut.RLock()
	valid, ok = bls.popCache[key.String()]
	bls.mut.RUnlock()
	if ok {
		return valid
	}

	proof, err := bls12.NewG2().FromCompressed([]byte(popBytes))
	if err != nil {
		return false
	}

	valid = bls.popVerify(replica.PublicKey().(*PublicKey), proof)

	bls.mut.Lock()
	bls.popCache[key.String()] = valid
	bls.mut.Unlock()

	return valid
}

func (bls *bls12Base) coreAggregateVerify(publicKeys []*PublicKey, messages [][]byte, signature *bls12.PointG2) bool {
	n := len(publicKeys)
	// validate input
	if n != len(messages) {
		return false
	}

	// precondition n >= 1
	if n < 1 {
		return false
	}

	if !bls.subgroupCheck(signature) {
		return false
	}

	engine := bls12.NewEngine()

	for i := 0; i < n; i++ {
		q, err := engine.G2.HashToCurve(messages[i], domain)
		if err != nil {
			return false
		}
		engine.AddPair(publicKeys[i].p, q)
	}

	engine.AddPairInv(&bls12.G1One, signature)
	return engine.Result().IsOne()
}

func (bls *bls12Base) aggregateVerify(publicKeys []*PublicKey, messages [][]byte, signature *bls12.PointG2) bool {
	set := make(map[string]struct{})
	for _, m := range messages {
		set[string(m)] = struct{}{}
	}
	return len(messages) == len(set) && bls.coreAggregateVerify(publicKeys, messages, signature)
}

func (bls *bls12Base) fastAggregateVerify(publicKeys []*PublicKey, message []byte, signature *bls12.PointG2) bool {
	engine := bls12.NewEngine()
	var aggregate bls12.PointG1
	for _, pk := range publicKeys {
		engine.G1.Add(&aggregate, &aggregate, pk.p)
	}
	return bls.coreVerify(&PublicKey{p: &aggregate}, message, signature, domain)
}

// Sign creates a cryptographic signature of the given message.
func (bls *bls12Base) Sign(message []byte) (signature types.QuorumSignature, err error) {
	p, err := bls.coreSign(message, domain)
	if err != nil {
		return nil, fmt.Errorf("bls12: coreSign failed: %w", err)
	}
	bf := bitfield.Bitfield{}
	//bf.Add(bls.opts.ID())
	bf.Add(bls.conf.Id)
	return &AggregateSignature{sig: *p, participants: bf}, nil
}

// Combine combines multiple signatures into a single signature.
func (bls *bls12Base) Combine(signatures ...types.QuorumSignature) (combined types.QuorumSignature, err error) {
	if len(signatures) < 2 {
		return nil, crypto.ErrCombineMultiple
	}

	g2 := bls12.NewG2()
	agg := bls12.PointG2{}
	var participants bitfield.Bitfield
	for _, sig1 := range signatures {
		if sig2, ok := sig1.(*AggregateSignature); ok {
			sig2.participants.RangeWhile(func(id types.ID) bool {
				if participants.Contains(id) {
					err = crypto.ErrCombineOverlap
					return false
				}
				participants.Add(id)
				return true
			})
			if err != nil {
				return nil, err
			}
			g2.Add(&agg, &agg, &sig2.sig)
		} else {
			log.Panicf("cannot combine incompatible signature types %T (expected %T)", sig1, sig2)
		}
	}
	return &AggregateSignature{sig: agg, participants: participants}, nil
}

// Verify verifies the given quorum signature against the message.
func (bls *bls12Base) Verify(signature types.QuorumSignature, message []byte) bool {
	s, ok := signature.(*AggregateSignature)
	if !ok {
		log.Panicf("cannot verify signature of incompatible types %T (expected %T)", signature, s)
	}

	n := s.Participants().Len()

	if n == 1 {
		id := firstParticipant(s.Participants())
		pk, ok := bls.publicKey(id)
		if !ok {
			log.Warnf("Missing public key for ID %d", id)
			return false
		}
		return bls.coreVerify(pk, message, &s.sig, domain)
	}

	// else if l > 1:
	pks := make([]*PublicKey, 0, n)
	s.Participants().RangeWhile(func(id types.ID) bool {
		pk, ok := bls.publicKey(id)
		if ok {
			pks = append(pks, pk)
			return true
		}
		log.Warnf("Missing public key for ID %d", id)
		return false
	})
	if len(pks) != n {
		return false
	}
	return bls.fastAggregateVerify(pks, message, &s.sig)
}

// BatchVerify verifies the given quorum signature against the batch of messages.
func (bls *bls12Base) BatchVerify(signature types.QuorumSignature, batch map[types.ID][]byte) bool {
	s, ok := signature.(*AggregateSignature)
	if !ok {
		log.Panicf("cannot verify incompatible signature types %T (expected %T)", signature, s)
	}

	if s.Participants().Len() != len(batch) {
		return false
	}

	pks := make([]*PublicKey, 0, len(batch))
	msgs := make([][]byte, 0, len(batch))

	for id, msg := range batch {
		msgs = append(msgs, msg)
		pk, ok := bls.publicKey(id)
		if !ok {
			log.Warnf("Missing public key for ID %d", id)
			return false
		}
		pks = append(pks, pk)
	}

	if len(batch) == 1 {
		return bls.coreVerify(pks[0], msgs[0], &s.sig, domain)
	}

	return bls.aggregateVerify(pks, msgs, &s.sig)
}

///////////////////

// GenerateECDSAPrivateKey returns a new ECDSA private key.
func GenerateECDSAPrivateKey() (pk *ecdsa.PrivateKey, err error) {
	pk, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

// GenerateED25519Key generates ed25519 key.
func GenerateED25519Key() (pub ed25519.PublicKey, pk ed25519.PrivateKey, err error) {
	pub, pk, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return pub, pk, nil
}

// GenerateRootCert generates a self-signed TLS certificate to act as a CA.
func GenerateRootCert(privateKey *ecdsa.PrivateKey) (cert *x509.Certificate, err error) {
	sn, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	caTmpl := &x509.Certificate{
		SerialNumber:          sn,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(caBytes)
}

// GenerateTLSCert generates a TLS certificate for the server that is valid for the given hosts.
func GenerateTLSCert(id types.ID, hosts []string, parent *x509.Certificate, signeeKey *ecdsa.PublicKey, signerKey *ecdsa.PrivateKey) (cert *x509.Certificate, err error) {
	sn, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	caTmpl := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("%d", id),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			caTmpl.IPAddresses = append(caTmpl.IPAddresses, ip)
		} else {
			caTmpl.DNSNames = append(caTmpl.DNSNames, h)
		}
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTmpl, parent, signeeKey, signerKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(caBytes)
}

// PrivateKeyToPEM encodes the private key in PEM format.
func PrivateKeyToPEM(key types.PrivateKey) ([]byte, error) {
	var (
		marshaled []byte
		keyType   string
		//err       error
	)
	switch k := key.(type) {
	case *PrivateKey:
		marshaled = k.ToBytes()
		keyType = PrivateKeyFileType
	}

	b := &pem.Block{
		Type:  keyType,
		Bytes: marshaled,
	}
	return pem.EncodeToMemory(b), nil
}

// WritePrivateKeyFile writes a private key to the specified file.
func WritePrivateKeyFile(key types.PrivateKey, filePath string) (err error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	b, err := PrivateKeyToPEM(key)
	if err != nil {
		return
	}

	_, err = f.Write(b)
	return
}

// PublicKeyToPEM encodes the public key in PEM format.
func PublicKeyToPEM(key types.PublicKey) ([]byte, error) {
	var (
		marshaled []byte
		keyType   string
		//err       error
	)
	switch k := key.(type) {
	case *PublicKey:
		marshaled = k.ToBytes()
		keyType = PublicKeyFileType
	}

	b := &pem.Block{
		Type:  keyType,
		Bytes: marshaled,
	}
	return pem.EncodeToMemory(b), nil
}

// WritePublicKeyFile writes a public key to the specified file.
func WritePublicKeyFile(key types.PublicKey, filePath string) (err error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}

	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	b, err := PublicKeyToPEM(key)
	if err != nil {
		return err
	}

	_, err = f.Write(b)
	return err
}

// WriteCertFile writes an x509 certificate to a file.
func WriteCertFile(cert *x509.Certificate, file string) (err error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}

	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	b := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.Encode(f, b)
}

// ParsePrivateKey parses a PEM encoded private key.
func ParsePrivateKey(buf []byte) (key types.PrivateKey, err error) {
	b, _ := pem.Decode(buf)
	switch b.Type {
	case PrivateKeyFileType:
		k := &PrivateKey{}
		k.FromBytes(b.Bytes)
		key = k
	default:
		return nil, fmt.Errorf("private key file type did not match any known types %v", b.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}
	return
}

// ReadPrivateKeyFile reads a private key from the specified file.
func ReadPrivateKeyFile(keyFile string) (key types.PrivateKey, err error) {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return ParsePrivateKey(b)
}

// ParsePublicKey parses a PEM encoded public key
func ParsePublicKey(buf []byte) (key types.PublicKey, err error) {
	b, _ := pem.Decode(buf)
	if b == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	switch b.Type {
	case PublicKeyFileType:
		k := &PublicKey{}
		err = k.FromBytes(b.Bytes)
		if err != nil {
			return nil, err
		}
		key = k
	default:
		return nil, fmt.Errorf("public key file type did not match any known types %v", b.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}
	return
}

// ReadPublicKeyFile reads a public key from the specified file.
func ReadPublicKeyFile(keyFile string) (key types.PublicKey, err error) {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return ParsePublicKey(b)
}

// ReadCertFile read an x509 certificate from a file.
func ReadCertFile(certFile string) (cert *x509.Certificate, err error) {
	d, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode(d)
	if b == nil {
		return nil, fmt.Errorf("failed to decode key")
	}

	if b.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("file type did not match")
	}

	cert, err = x509.ParseCertificate(b.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// KeyChain contains the keys and certificates needed by a replica, in PEM format.
type KeyChain struct {
	PrivateKey     []byte
	PublicKey      []byte
	Certificate    []byte
	CertificateKey []byte
}

// GenerateKeyChain generates keys and certificates for a replica.
func GenerateKeyChain(id types.ID, validFor []string, crypto string, ca *x509.Certificate, caKey *ecdsa.PrivateKey) (KeyChain, error) {
	ecdsaKey, err := GenerateECDSAPrivateKey()
	if err != nil {
		return KeyChain{}, err
	}
	certKeyPEM, err := PrivateKeyToPEM(ecdsaKey)
	if err != nil {
		return KeyChain{}, err
	}

	cert, err := GenerateTLSCert(id, validFor, ca, &ecdsaKey.PublicKey, caKey)
	if err != nil {
		return KeyChain{}, err
	}

	certPEM := CertToPEM(cert)

	var privateKey types.PrivateKey
	var publicKey types.PublicKey
	switch crypto {
	case "bls12":
		privateKey, err = GeneratePrivateKey()
		if err != nil {
			return KeyChain{}, fmt.Errorf("failed to generate bls12-381 private key: %w", err)
		}
		publicKey = privateKey.Public()
	default:
		return KeyChain{}, fmt.Errorf("unknown crypto implementation: %s", crypto)
	}

	privateKeyPEM, err := PrivateKeyToPEM(privateKey)
	if err != nil {
		return KeyChain{}, err
	}

	publicKeyPEM, err := PublicKeyToPEM(publicKey)
	if err != nil {
		return KeyChain{}, err
	}

	WritePrivateKeyFile(privateKey, "/Users/huey/Documents/cs/project/code/hxy352/certs/"+fmt.Sprintf("%d", id)+".key")
	WritePublicKeyFile(publicKey, "/Users/huey/Documents/cs/project/code/hxy352/certs/"+fmt.Sprintf("%d", id)+".pub")
	WriteCertFile(cert, "/Users/huey/Documents/cs/project/code/hxy352/certs/"+fmt.Sprintf("%d", id)+".cert")
	//WriteCertFile(ecdsaKey, "/Users/huey/Documents/cs/project/code/hxy352/certs/" + fmt.Sprintf("%d", id) + ".cert")

	return KeyChain{
		PrivateKey:     privateKeyPEM,
		PublicKey:      publicKeyPEM,
		Certificate:    certPEM,
		CertificateKey: certKeyPEM,
	}, nil
}

// GenerateCA returns a certificate authority for generating new certificates.
func GenerateCA() (pk *ecdsa.PrivateKey, ca *x509.Certificate, err error) {
	pk, err = GenerateECDSAPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate signing key: %w", err)
	}
	ca, err = GenerateRootCert(pk)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate root certificate: %w", err)
	}
	return pk, ca, nil
}

// CertToPEM encodes an x509 certificate in PEM format.
func CertToPEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}
