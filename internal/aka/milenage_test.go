package aka

import (
	"encoding/hex"
	"testing"

	"aka-server/internal/model"

	"github.com/wmnsk/milenage"
)

func TestGenerateVector(t *testing.T) {
	sub := &model.Subscriber{
		IMSI: "123456789012345",
		Ki:   "00112233445566778899aabbccddeeff",
		Opc:  "000102030405060708090a0b0c0d0e0f",
		SQN:  "000000000020", // SEQ=1, IND=0
		AMF:  "8000",
	}

	vec, newSQN, err := GenerateVector(sub)
	if err != nil {
		t.Fatalf("GenerateVector failed: %v", err)
	}

	if len(vec.Rand) != 32 {
		t.Errorf("Invalid RAND length: %d", len(vec.Rand))
	}
	if len(vec.Autn) != 32 {
		t.Errorf("Invalid AUTN length: %d", len(vec.Autn))
	}

	// SQN should be incremented.
	// Old SQN: 0x20 (32). SEQ=1.
	// New SQN should be SEQ=2 -> 0x40 (64).
	if newSQN != "000000000040" {
		t.Errorf("Expected new SQN 000000000040, got %s", newSQN)
	}
}

func TestResync(t *testing.T) {
	// Scenario: HE has SQN=10. USIM has SQN=20.
	// HE sends vector with SQN=11.
	// USIM rejects. Sends AUTS with SQN=20.
	// HE recovers SQN=20, updates to 21, generates new vector.

	kiHex := "00112233445566778899aabbccddeeff"
	opcHex := "000102030405060708090a0b0c0d0e0f"
	amfHex := "8000"

	ki, _ := hex.DecodeString(kiHex)
	opc, _ := hex.DecodeString(opcHex)
	// amf, _ := hex.DecodeString(amfHex)
	// amfVal := binary.BigEndian.Uint16(amf)

	// 1. Prepare HE state
	sub := &model.Subscriber{
		IMSI: "123456789012345",
		Ki:   kiHex,
		Opc:  opcHex,
		SQN:  "000000000140", // SQN=10 (SEQ=10, IND=0) -> 10 << 5 = 320 = 0x140
		AMF:  amfHex,
	}

	// 2. Simulate USIM generating AUTS
	// USIM has SQN=20 (SEQ=20, IND=0) -> 20 << 5 = 640 = 0x280
	usimSqn := uint64(0x280)

	// We need a RAND. Let's say HE sent some RAND.
	randHex := "00000000000000000000000000000000"
	randBytes, _ := hex.DecodeString(randHex)

	// USIM generates AUTS
	// NewWithOPc(k, opc, rand, sqn, amf)
	// Note: For GenerateAUTS, the SQN passed to New should be the USIM's SQN.
	m := milenage.NewWithOPc(ki, opc, randBytes, usimSqn, 0) // AMF is not used for AUTS generation? Wait.
	// F1* uses AMF=0. F5* uses RAND.

	autsBytes, err := m.GenerateAUTS()
	if err != nil {
		t.Fatalf("Failed to generate AUTS: %v", err)
	}
	autsHex := hex.EncodeToString(autsBytes)

	// 3. Call Resync on HE
	vec, newSQN, err := Resync(sub, randHex, autsHex)
	if err != nil {
		t.Fatalf("Resync failed: %v", err)
	}

	// HE should have recovered SQN=20 (0x280).
	// And then GenerateVector should have incremented it to 21 (0x2A0).
	// 21 << 5 = 672 = 0x2A0.

	expectedSQN := "0000000002a0"
	if newSQN != expectedSQN {
		t.Errorf("Expected new SQN %s, got %s", expectedSQN, newSQN)
	}

	if len(vec.Autn) != 32 {
		t.Errorf("Invalid AUTN length in resync vector")
	}
}
