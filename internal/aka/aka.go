// Package aka は Milenage による AKA 認証ベクターの生成と再同期の検証を行う。
package aka

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wmnsk/milenage"
)

// ErrAUTSInvalid は AUTS の MAC-S が一致しなかったことを表す。
var ErrAUTSInvalid = errors.New("AUTS verification failed")

// Credentials は加入者の鍵情報。
type Credentials struct {
	Ki  []byte // 16 バイト
	OPc []byte // 16 バイト
	AMF uint16
}

// Vector は AKA 認証ベクター。
type Vector struct {
	RAND []byte
	XRES []byte
	AUTN []byte
	CK   []byte
	IK   []byte
}

// NewRAND は暗号論的乱数で RAND を生成する。
func NewRAND() []byte {
	b := make([]byte, 16)
	rand.Read(b)
	return b
}

func newMilenage(c Credentials, rnd []byte, sqn uint64) *milenage.Milenage {
	return milenage.NewWithOPc(c.Ki, c.OPc, rnd, sqn, c.AMF)
}

// Generate は指定した RAND と SQN から認証ベクターを生成する。
func Generate(c Credentials, rnd []byte, sqn uint64) (Vector, error) {
	m := newMilenage(c, rnd, sqn)
	if err := m.ComputeAll(); err != nil {
		return Vector{}, fmt.Errorf("milenage: %w", err)
	}
	autn, err := m.GenerateAUTN()
	if err != nil {
		return Vector{}, fmt.Errorf("milenage: %w", err)
	}
	return Vector{RAND: rnd, XRES: m.RES, AUTN: autn, CK: m.CK, IK: m.IK}, nil
}

// VerifyAUTS は再同期トークン AUTS を検証し、端末側の SQN（SQN_MS）を返す。
// rnd は、同期失敗を起こした認証で使った RAND。
// MAC-S が一致しなければ ErrAUTSInvalid を返す。
func VerifyAUTS(c Credentials, rnd, auts []byte) (uint64, error) {
	if len(auts) != 14 {
		return 0, fmt.Errorf("AUTS must be 14 bytes, got %d", len(auts))
	}
	m := newMilenage(c, rnd, 0)
	aks, err := m.F5Star()
	if err != nil {
		return 0, fmt.Errorf("milenage: %w", err)
	}
	sqn := make([]byte, 6)
	subtle.XORBytes(sqn, auts[:6], aks)

	// MAC-S は AMF を 0x0000 として計算する（TS 33.102 6.3.3）。
	macS, err := m.F1Star(sqn, []byte{0x00, 0x00})
	if err != nil {
		return 0, fmt.Errorf("milenage: %w", err)
	}
	if !hmac.Equal(macS, auts[6:]) {
		return 0, ErrAUTSInvalid
	}
	return binary.BigEndian.Uint64(append([]byte{0, 0}, sqn...)), nil
}

// DerivePrime は EAP-AKA' 用の CK' と IK' を導出する（TS 33.402 Annex A.2、RFC 5448）。
// AUTN の先頭 6 バイト（SQN xor AK）を鍵導出に使う。
func DerivePrime(v Vector, networkName string) (ckPrime, ikPrime []byte) {
	key := make([]byte, 0, 32)
	key = append(key, v.CK...)
	key = append(key, v.IK...)

	s := make([]byte, 0, 1+len(networkName)+2+6+2)
	s = append(s, 0x20) // FC
	s = append(s, networkName...)
	s = binary.BigEndian.AppendUint16(s, uint16(len(networkName)))
	s = append(s, v.AUTN[:6]...)
	s = binary.BigEndian.AppendUint16(s, 6)

	mac := hmac.New(sha256.New, key)
	mac.Write(s)
	out := mac.Sum(nil)
	return out[:16], out[16:]
}
