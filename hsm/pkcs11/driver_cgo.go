//go:build cgo

package pkcs11

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/miekg/pkcs11"
)

type Mechanism string

const (
	MechanismECDSASHA256    Mechanism = "ECDSA-SHA256"
	MechanismECDSASHA384    Mechanism = "ECDSA-SHA384"
	MechanismECDSASHA512    Mechanism = "ECDSA-SHA512"
	MechanismRSAPKCS1SHA256 Mechanism = "RSA-PKCS1-SHA256"
	MechanismRSAPSSSHA256   Mechanism = "RSA-PSS-SHA256"
	MechanismEd25519        Mechanism = "Ed25519"
)

type KeySpec struct {
	Label       string
	ID          []byte
	Mechanism   Mechanism
	Curve       string // P-256, P-384, P-521
	Bits        int    // RSA bits
	Extractable bool
}

type KeyID struct {
	Label string
	ID    []byte
}

type KeyInfo struct {
	ID           KeyID
	Algorithm    string
	PublicKeyPEM string
}

type SlotInfo struct {
	SlotID          uint
	SlotDescription string
	TokenLabel      string
	ManufacturerID  string
	Model           string
}

type Signer interface {
	crypto.Signer
	KeyID() KeyID
}

// Driver implements industrial HSM via PKCS#11 for single device.
type Driver struct {
	ctx         *pkcs11.Ctx
	cfg         Config
	slotID      uint
	sessionPool chan pkcs11.SessionHandle
	mu          sync.Mutex
	closed      bool
}

// NewDriver initializes PKCS#11 module and logs into token.
func NewDriver(cfg Config) (*Driver, error) {
	if cfg.LibraryPath == "" {
		return nil, errors.New("pkcs11: LibraryPath required")
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = 4
	}
	p := pkcs11.New(cfg.LibraryPath)
	if p == nil {
		return nil, fmt.Errorf("pkcs11: failed to load %s", cfg.LibraryPath)
	}
	if err := p.Initialize(); err != nil {
		return nil, fmt.Errorf("pkcs11 Initialize: %w", err)
	}
	slots, err := p.GetSlotList(true)
	if err != nil {
		p.Destroy()
		p.Finalize()
		return nil, fmt.Errorf("GetSlotList: %w", err)
	}
	if len(slots) == 0 {
		p.Destroy()
		p.Finalize()
		return nil, errors.New("pkcs11: no slots with token found")
	}
	var slotID uint
	if cfg.SlotID != nil {
		slotID = *cfg.SlotID
	} else if cfg.TokenLabel != "" {
		found := false
		for _, s := range slots {
			info, err := p.GetTokenInfo(s)
			if err != nil {
				continue
			}
			if info.Label == cfg.TokenLabel {
				slotID = s
				found = true
				break
			}
		}
		if !found {
			p.Destroy()
			p.Finalize()
			return nil, fmt.Errorf("pkcs11: token %q not found", cfg.TokenLabel)
		}
	} else {
		slotID = slots[0]
	}

	d := &Driver{ctx: p, cfg: cfg, slotID: slotID, sessionPool: make(chan pkcs11.SessionHandle, cfg.MaxSessions)}
	// Pre-open sessions and login
	for i := 0; i < cfg.MaxSessions; i++ {
		sh, err := p.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
		if err != nil {
			d.Close()
			return nil, fmt.Errorf("OpenSession: %w", err)
		}
		if cfg.PIN != "" {
			if err := p.Login(sh, pkcs11.CKU_USER, cfg.PIN); err != nil && err != pkcs11.Error(pkcs11.CKR_USER_ALREADY_LOGGED_IN) {
				p.CloseSession(sh)
				d.Close()
				return nil, fmt.Errorf("Login: %w", err)
			}
		}
		d.sessionPool <- sh
	}
	return d, nil
}

func (d *Driver) getSession(ctx context.Context) (pkcs11.SessionHandle, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case sh := <-d.sessionPool:
		return sh, nil
	}
}

func (d *Driver) putSession(sh pkcs11.SessionHandle) {
	select {
	case d.sessionPool <- sh:
	default:
		d.ctx.CloseSession(sh)
	}
}

func mechanismFor(mech Mechanism) ([]*pkcs11.Mechanism, error) {
	switch mech {
	case MechanismECDSASHA256, MechanismECDSASHA384, MechanismECDSASHA512, "ECDSA-P256", "ECDSA-P384", "":
		return []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}, nil
	case MechanismRSAPKCS1SHA256:
		return []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)}, nil
	case MechanismRSAPSSSHA256:
		return []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_PSS, nil)}, nil
	default:
		return nil, fmt.Errorf("unsupported mechanism %q", mech)
	}
}

func ecParams(curve string) []byte {
	switch curve {
	case "P-384":
		return []byte{0x06, 0x05, 0x2B, 0x81, 0x04, 0x00, 0x22} // secp384r1 OID 1.3.132.0.34
	case "P-521":
		return []byte{0x06, 0x05, 0x2B, 0x81, 0x04, 0x00, 0x23} // secp521r1
	default: // P-256
		return []byte{0x06, 0x08, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07} // prime256v1
	}
}

func curveFromParams(params []byte) elliptic.Curve {
	hex := fmt.Sprintf("%x", params)
	switch hex {
	case "06052b81040022":
		return elliptic.P384()
	case "06052b81040023":
		return elliptic.P521()
	default:
		return elliptic.P256()
	}
}

// GenerateKey creates key pair inside HSM.
func (d *Driver) GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error) {
	if spec.Label == "" {
		return nil, errors.New("Label required")
	}
	id := spec.ID
	if len(id) == 0 {
		id = []byte(spec.Label)
	}
	sh, err := d.getSession(ctx)
	if err != nil {
		return nil, err
	}
	defer d.putSession(sh)

	mech := spec.Mechanism
	if mech == "" {
		mech = MechanismECDSASHA256
	}

	// Default RSA handling
	if mech == MechanismRSAPKCS1SHA256 || mech == MechanismRSAPSSSHA256 {
		bits := spec.Bits
		if bits == 0 {
			bits = 2048
		}
		pubTemplate := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, spec.Label),
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
			pkcs11.NewAttribute(pkcs11.CKA_MODULUS_BITS, bits),
			pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, []byte{1, 0, 1}),
		}
		privTemplate := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, spec.Label),
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, spec.Extractable),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		}
		_, _, err := d.ctx.GenerateKeyPair(sh,
			[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_KEY_PAIR_GEN, nil)},
			pubTemplate, privTemplate)
		if err != nil {
			return nil, fmt.Errorf("GenerateKeyPair RSA: %w", err)
		}
		pub, pemStr, err := d.getRSAPublicKey(sh, spec.Label, id)
		if err != nil {
			return nil, err
		}
		_ = pub
		return &KeyInfo{ID: KeyID{Label: spec.Label, ID: id}, Algorithm: fmt.Sprintf("RSA-%d", bits), PublicKeyPEM: pemStr}, nil
	}

	// ECDSA
	curve := spec.Curve
	if curve == "" {
		curve = "P-256"
	}
	pubTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, spec.Label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, ecParams(curve)),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
	}
	privTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, spec.Label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, spec.Extractable),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
	}
	pubHandle, _, err := d.ctx.GenerateKeyPair(sh,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_EC_KEY_PAIR_GEN, nil)},
		pubTemplate, privTemplate)
	if err != nil {
		return nil, fmt.Errorf("GenerateKeyPair EC: %w", err)
	}
	attrs, err := d.ctx.GetAttributeValue(sh, pubHandle, []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, nil),
	})
	if err != nil {
		return nil, fmt.Errorf("GetAttributeValue EC_POINT: %w", err)
	}
	var ecPoint, params []byte
	for _, a := range attrs {
		if a.Type == pkcs11.CKA_EC_POINT {
			ecPoint = a.Value
		}
		if a.Type == pkcs11.CKA_EC_PARAMS {
			params = a.Value
		}
	}
	pub, pemStr, err := ecPointToPEM(ecPoint, params)
	if err != nil {
		return nil, err
	}
	_ = pub
	algo := "ECDSA-" + curve
	return &KeyInfo{ID: KeyID{Label: spec.Label, ID: id}, Algorithm: algo, PublicKeyPEM: pemStr}, nil
}

func ecPointToPEM(ecPoint, params []byte) (crypto.PublicKey, string, error) {
	// CKA_EC_POINT is DER OCTET STRING wrapping uncompressed point
	var pointBytes []byte
	if len(ecPoint) == 0 {
		return nil, "", errors.New("empty EC_POINT")
	}
	// Try ASN.1 octet string decode, fallback to raw
	var octet []byte
	if _, err := asn1.Unmarshal(ecPoint, &octet); err == nil {
		pointBytes = octet
	} else {
		// Some tokens return raw point
		pointBytes = ecPoint
		// If first byte is 0x04 (uncompressed), keep; if wrapped, strip 04?
		if len(pointBytes) > 2 && pointBytes[0] == 0x04 && pointBytes[1] == 0x04 {
			pointBytes = pointBytes[1:]
		}
	}
	curve := curveFromParams(params)
	if len(pointBytes) == 0 || pointBytes[0] != 0x04 {
		return nil, "", fmt.Errorf("unexpected EC point format %x", pointBytes[:4])
	}
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(pointBytes) != 1+2*byteLen {
		return nil, "", fmt.Errorf("invalid EC point length %d", len(pointBytes))
	}
	x := new(big.Int).SetBytes(pointBytes[1 : 1+byteLen])
	y := new(big.Int).SetBytes(pointBytes[1+byteLen:])
	pub := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, "", err
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return pub, pemStr, nil
}

func (d *Driver) getRSAPublicKey(sh pkcs11.SessionHandle, label string, id []byte) (crypto.PublicKey, string, error) {
	tmpl := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
	}
	if err := d.ctx.FindObjectsInit(sh, tmpl); err != nil {
		return nil, "", err
	}
	objs, _, err := d.ctx.FindObjects(sh, 1)
	if err != nil {
		d.ctx.FindObjectsFinal(sh)
		return nil, "", err
	}
	d.ctx.FindObjectsFinal(sh)
	if len(objs) == 0 {
		return nil, "", fmt.Errorf("public key not found %s", label)
	}
	attrs, err := d.ctx.GetAttributeValue(sh, objs[0], []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_MODULUS, nil),
		pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, nil),
	})
	if err != nil {
		return nil, "", err
	}
	var mod, exp []byte
	for _, a := range attrs {
		if a.Type == pkcs11.CKA_MODULUS {
			mod = a.Value
		}
		if a.Type == pkcs11.CKA_PUBLIC_EXPONENT {
			exp = a.Value
		}
	}
	n := new(big.Int).SetBytes(mod)
	e := int(new(big.Int).SetBytes(exp).Int64())
	pub := &rsa.PublicKey{N: n, E: e}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, "", err
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return pub, pemStr, nil
}

func (d *Driver) findPrivateKey(ctx context.Context, id KeyID) (pkcs11.ObjectHandle, error) {
	sh, err := d.getSession(ctx)
	if err != nil {
		return 0, err
	}
	defer d.putSession(sh)
	return d.findPrivateKeyWithSession(sh, id)
}

func (d *Driver) findPrivateKeyWithSession(sh pkcs11.SessionHandle, id KeyID) (pkcs11.ObjectHandle, error) {
	var tmpl []*pkcs11.Attribute
	if id.Label != "" {
		tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_LABEL, id.Label))
	}
	if len(id.ID) > 0 {
		tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_ID, id.ID))
	}
	tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY))
	if err := d.ctx.FindObjectsInit(sh, tmpl); err != nil {
		return 0, err
	}
	objs, _, err := d.ctx.FindObjects(sh, 1)
	if err != nil {
		d.ctx.FindObjectsFinal(sh)
		return 0, err
	}
	d.ctx.FindObjectsFinal(sh)
	if len(objs) == 0 {
		return 0, fmt.Errorf("private key not found %v", id)
	}
	return objs[0], nil
}

func (d *Driver) GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error) {
	sh, err := d.getSession(ctx)
	if err != nil {
		return nil, "", err
	}
	defer d.putSession(sh)

	var tmpl []*pkcs11.Attribute
	if id.Label != "" {
		tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_LABEL, id.Label))
	}
	if len(id.ID) > 0 {
		tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_ID, id.ID))
	}
	tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY))
	if err := d.ctx.FindObjectsInit(sh, tmpl); err != nil {
		return nil, "", err
	}
	objs, _, err := d.ctx.FindObjects(sh, 2)
	if err != nil {
		d.ctx.FindObjectsFinal(sh)
		return nil, "", err
	}
	d.ctx.FindObjectsFinal(sh)
	if len(objs) == 0 {
		return nil, "", fmt.Errorf("public key not found %v", id)
	}
	// Try EC first: EC keys have CKA_EC_POINT, RSA do not. Separate calls to avoid CKR_ATTRIBUTE_TYPE_INVALID on RSA.
	if ecAttrs, err := d.ctx.GetAttributeValue(sh, objs[0], []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, nil),
	}); err == nil {
		var ecPoint, params []byte
		for _, a := range ecAttrs {
			if a.Type == pkcs11.CKA_EC_POINT {
				ecPoint = a.Value
			}
			if a.Type == pkcs11.CKA_EC_PARAMS {
				params = a.Value
			}
		}
		if len(ecPoint) > 0 {
			return ecPointToPEM(ecPoint, params)
		}
	}
	// Fallback to RSA
	return d.getRSAPublicKey(sh, id.Label, id.ID)
}

// Sign signs a pre-hashed digest. For ECDSA, PKCS#11 CKM_ECDSA expects raw digest and returns raw r||s; we convert to ASN.1 DER.
func (d *Driver) Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error) {
	if len(digest) == 0 {
		return nil, errors.New("digest empty")
	}
	sh, err := d.getSession(ctx)
	if err != nil {
		return nil, err
	}
	defer d.putSession(sh)

	priv, err := d.findPrivateKeyWithSession(sh, id)
	if err != nil {
		return nil, err
	}
	mechs, err := mechanismFor(mech)
	if err != nil {
		return nil, err
	}
	// For CKM_ECDSA, digest length determines curve; for RSA, same mech works for pre-hashed
	if err := d.ctx.SignInit(sh, mechs, priv); err != nil {
		return nil, fmt.Errorf("SignInit: %w", err)
	}
	sig, err := d.ctx.Sign(sh, digest)
	if err != nil {
		return nil, fmt.Errorf("Sign: %w", err)
	}
	// If ECDSA, convert raw r||s to ASN.1
	if mech == MechanismECDSASHA256 || mech == MechanismECDSASHA384 || mech == MechanismECDSASHA512 || mech == "" {
		if len(sig)%2 == 0 && len(sig) >= 64 {
			half := len(sig) / 2
			r := new(big.Int).SetBytes(sig[:half])
			s := new(big.Int).SetBytes(sig[half:])
			return asn1.Marshal(struct {
				R, S *big.Int
			}{R: r, S: s})
		}
	}
	return sig, nil
}

type pkcs11Signer struct {
	driver *Driver
	keyID  KeyID
	mech   Mechanism
	pub    crypto.PublicKey
}

func (s *pkcs11Signer) Public() crypto.PublicKey { return s.pub }
func (s *pkcs11Signer) KeyID() KeyID             { return s.keyID }
func (s *pkcs11Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// opts.HashFunc indicates hash; digest is already hashed
	if opts != nil && opts.HashFunc() != 0 {
		// digest already hashed by caller (e.g. crypto.SHA256)
	}
	return s.driver.Sign(context.Background(), s.keyID, digest, s.mech)
}

func (d *Driver) Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error) {
	pub, _, err := d.GetPublicKey(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pkcs11Signer{driver: d, keyID: id, mech: mech, pub: pub}, nil
}

func (d *Driver) DeleteKey(ctx context.Context, id KeyID) error {
	sh, err := d.getSession(ctx)
	if err != nil {
		return err
	}
	defer d.putSession(sh)

	for _, class := range []uint{pkcs11.CKO_PRIVATE_KEY, pkcs11.CKO_PUBLIC_KEY} {
		var tmpl []*pkcs11.Attribute
		if id.Label != "" {
			tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_LABEL, id.Label))
		}
		if len(id.ID) > 0 {
			tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_ID, id.ID))
		}
		tmpl = append(tmpl, pkcs11.NewAttribute(pkcs11.CKA_CLASS, class))
		if err := d.ctx.FindObjectsInit(sh, tmpl); err != nil {
			continue
		}
		objs, _, err := d.ctx.FindObjects(sh, 10)
		d.ctx.FindObjectsFinal(sh)
		if err != nil {
			continue
		}
		for _, o := range objs {
			if err := d.ctx.DestroyObject(sh, o); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Driver) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	sh, err := d.getSession(ctx)
	if err != nil {
		return nil, err
	}
	defer d.putSession(sh)

	tmpl := []*pkcs11.Attribute{pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY)}
	if err := d.ctx.FindObjectsInit(sh, tmpl); err != nil {
		return nil, err
	}
	objs, _, err := d.ctx.FindObjects(sh, 100)
	if err != nil {
		d.ctx.FindObjectsFinal(sh)
		return nil, err
	}
	d.ctx.FindObjectsFinal(sh)
	var out []KeyInfo
	for _, o := range objs {
		attrs, err := d.ctx.GetAttributeValue(sh, o, []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, nil),
			pkcs11.NewAttribute(pkcs11.CKA_ID, nil),
			pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, nil),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, nil),
		})
		if err != nil {
			continue
		}
		var label string
		var id []byte
		var params []byte
		var keyType []byte
		for _, a := range attrs {
			switch a.Type {
			case pkcs11.CKA_LABEL:
				label = string(a.Value)
			case pkcs11.CKA_ID:
				id = a.Value
			case pkcs11.CKA_EC_PARAMS:
				params = a.Value
			case pkcs11.CKA_KEY_TYPE:
				keyType = a.Value
			}
		}
		algo := "unknown"
		if len(params) > 0 {
			curve := "P-256"
			if fmt.Sprintf("%x", params) == "06052b81040022" {
				curve = "P-384"
			} else if fmt.Sprintf("%x", params) == "06052b81040023" {
				curve = "P-521"
			}
			algo = "ECDSA-" + curve
		} else if len(keyType) > 0 {
			algo = "RSA"
		}
		// fetch PEM
		_, pemStr, _ := d.GetPublicKey(ctx, KeyID{Label: label, ID: id})
		out = append(out, KeyInfo{ID: KeyID{Label: label, ID: id}, Algorithm: algo, PublicKeyPEM: pemStr})
	}
	return out, nil
}

func (d *Driver) Info(ctx context.Context) (*SlotInfo, error) {
	_ = ctx
	info, err := d.ctx.GetSlotInfo(d.slotID)
	if err != nil {
		return nil, err
	}
	tInfo, err := d.ctx.GetTokenInfo(d.slotID)
	if err != nil {
		return nil, err
	}
	return &SlotInfo{
		SlotID:          d.slotID,
		SlotDescription: info.SlotDescription,
		TokenLabel:      tInfo.Label,
		ManufacturerID:  tInfo.ManufacturerID,
		Model:           tInfo.Model,
	}, nil
}

func (d *Driver) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()
	close(d.sessionPool)
	for sh := range d.sessionPool {
		d.ctx.Logout(sh)
		d.ctx.CloseSession(sh)
	}
	d.ctx.Finalize()
	d.ctx.Destroy()
	return nil
}

// Ensure Driver satisfies crypto.Signer via Signer
var _ = crypto.Signer(&pkcs11Signer{})

// Helper for testing: generate digest sign verify locally
func init() {
	_ = rand.Reader
}
