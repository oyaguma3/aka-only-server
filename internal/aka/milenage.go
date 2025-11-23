package aka

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"aka-server/internal/model"

	"github.com/wmnsk/milenage"
)

type AuthVector struct {
	Rand string `json:"rand"`
	Autn string `json:"autn"`
	Xres string `json:"xres"`
	Ck   string `json:"ck"`
	Ik   string `json:"ik"`
}

// GenerateVector generates an authentication vector for the given subscriber.
// It returns the vector and the new SQN (hex string) to be updated in the DB.
func GenerateVector(sub *model.Subscriber) (*AuthVector, string, error) {
	ki, err := hex.DecodeString(sub.Ki)
	if err != nil {
		return nil, "", fmt.Errorf("invalid Ki: %w", err)
	}
	opc, err := hex.DecodeString(sub.Opc)
	if err != nil {
		return nil, "", fmt.Errorf("invalid OPC: %w", err)
	}
	sqnBytes, err := hex.DecodeString(sub.SQN)
	if err != nil {
		return nil, "", fmt.Errorf("invalid SQN: %w", err)
	}
	amfBytes, err := hex.DecodeString(sub.AMF)
	if err != nil {
		return nil, "", fmt.Errorf("invalid AMF: %w", err)
	}

	// Generate RAND
	randBytes := make([]byte, 16)
	_, err = rand.Read(randBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate RAND: %w", err)
	}

	// Increment SQN (SEQ only)
	sqnVal := binary.BigEndian.Uint64(append([]byte{0, 0}, sqnBytes...))
	ind := sqnVal & 0x1F
	seq := sqnVal >> 5
	seq++
	newSqnVal := (seq << 5) | ind

	newSqnBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(newSqnBytes, newSqnVal)
	newSqnBytes = newSqnBytes[2:] // Take last 6 bytes

	amfVal := binary.BigEndian.Uint16(amfBytes)

	// Calculate Milenage
	m := milenage.NewWithOPc(ki, opc, randBytes, newSqnVal, amfVal)

	if err := m.ComputeAll(); err != nil {
		return nil, "", fmt.Errorf("milenage computation failed: %w", err)
	}

	autn, err := m.GenerateAUTN()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate AUTN: %w", err)
	}

	vec := &AuthVector{
		Rand: hex.EncodeToString(randBytes),
		Autn: hex.EncodeToString(autn),
		Xres: hex.EncodeToString(m.RES),
		Ck:   hex.EncodeToString(m.CK),
		Ik:   hex.EncodeToString(m.IK),
	}

	return vec, hex.EncodeToString(newSqnBytes), nil
}

// Resync handles the resynchronization procedure.
// It verifies MAC-S in AUTS and recovers the SQN from the USIM.
// Returns the new vector and the recovered SQN.
func Resync(sub *model.Subscriber, randHex, autsHex string) (*AuthVector, string, error) {
	ki, err := hex.DecodeString(sub.Ki)
	if err != nil {
		return nil, "", fmt.Errorf("invalid Ki: %w", err)
	}
	opc, err := hex.DecodeString(sub.Opc)
	if err != nil {
		return nil, "", fmt.Errorf("invalid OPC: %w", err)
	}
	randBytes, err := hex.DecodeString(randHex)
	if err != nil {
		return nil, "", fmt.Errorf("invalid RAND: %w", err)
	}
	autsBytes, err := hex.DecodeString(autsHex)
	if err != nil {
		return nil, "", fmt.Errorf("invalid AUTS: %w", err)
	}
	if len(autsBytes) != 14 {
		return nil, "", fmt.Errorf("invalid AUTS length")
	}

	// AUTS = SQN_MS ^ AK* || MAC-S
	// SQN_MS ^ AK* is first 6 bytes
	// MAC-S is last 8 bytes
	sqnXorAk := autsBytes[:6]
	macS := autsBytes[6:]

	// Calculate AK* (AKS)
	// We use a dummy SQN/AMF for NewWithOPc because F5Star only depends on K, OPc, RAND
	m := milenage.NewWithOPc(ki, opc, randBytes, 0, 0)
	aks, err := m.F5Star()
	if err != nil {
		return nil, "", fmt.Errorf("failed to calculate AKS: %w", err)
	}

	// Recover SQN_MS
	sqnMsBytes := make([]byte, 6)
	for i := 0; i < 6; i++ {
		sqnMsBytes[i] = sqnXorAk[i] ^ aks[i]
	}

	// Verify MAC-S
	// MAC-S = F1*(K, RAND, SQN_MS, AMF=0)
	// We need to call F1Star.
	// F1Star takes sqn []byte and amf []byte
	amfStar := []byte{0, 0}
	xmacS, err := m.F1Star(sqnMsBytes, amfStar)
	if err != nil {
		return nil, "", fmt.Errorf("failed to calculate XMAC-S: %w", err)
	}

	if !bytes.Equal(macS, xmacS) {
		return nil, "", fmt.Errorf("MAC-S verification failed")
	}

	// Resync successful.
	// Update SQN to SQN_MS.
	// Then generate new vector.

	return GenerateVector(&model.Subscriber{
		IMSI: sub.IMSI,
		Ki:   sub.Ki,
		Opc:  sub.Opc,
		SQN:  hex.EncodeToString(sqnMsBytes),
		AMF:  sub.AMF,
	})
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
