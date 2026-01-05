package libatbus_message_handle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	libatbus "github.com/atframework/libatbus-go"
	impl "github.com/atframework/libatbus-go/impl"
	protocol "github.com/atframework/libatbus-go/protocol"
)

func TestGetBodyNameKnownAndUnknown(t *testing.T) {
	// Arrange
	known := int(protocol.MessageBody_EnMessageTypeID_CustomCommandReq)
	unknown := 999

	// Act
	gotKnown := GetBodyName(known)
	gotUnknown := GetBodyName(unknown)

	// Assert
	assert.Equal(t, "atframework.atbus.protocol.message_body.custom_command_req", gotKnown)
	assert.Equal(t, "Unknown", gotUnknown)
}

func TestMakeAccessDataPlaintextFromHandshakeWithoutPublicKey(t *testing.T) {
	// Arrange
	ad := &protocol.AccessData{Timestamp: 123, Nonce1: 7, Nonce2: 9}
	hd := &protocol.CryptoHandshakeData{Type: protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_SECP256R1, PublicKey: nil}

	// Act
	got := MakeAccessDataPlaintextFromHandshake(0x11, ad, hd)

	// Assert
	assert.Equal(t, "123:7-9:17", got)
}

func TestMakeAccessDataPlaintextFromHandshakeWithPublicKey(t *testing.T) {
	// Arrange
	pub := []byte("hello")
	h := sha256.Sum256(pub)
	expectedHash := hex.EncodeToString(h[:])
	ad := &protocol.AccessData{Timestamp: 123, Nonce1: 7, Nonce2: 9}
	hd := &protocol.CryptoHandshakeData{Type: protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_SECP384R1, PublicKey: pub}

	// Act
	got := MakeAccessDataPlaintextFromHandshake(0x11, ad, hd)

	// Assert
	assert.Equal(t, "123:7-9:17:3:"+expectedHash, got)
}

func TestMakeAccessDataPlaintextFromCustomCommand(t *testing.T) {
	// Arrange
	ad := &protocol.AccessData{Timestamp: 123, Nonce1: 7, Nonce2: 9}
	cs := &protocol.CustomCommandData{
		Commands: []*protocol.CustomCommandArgv{{Arg: []byte("a")}, {Arg: []byte("bc")}},
	}
	h := sha256.Sum256([]byte("abc"))
	expectedHash := hex.EncodeToString(h[:])

	// Act
	got := MakeAccessDataPlaintextFromCustomCommand(0x11, ad, cs)

	// Assert
	assert.Equal(t, "123:7-9:17:"+expectedHash, got)
}

func TestCalculateAccessDataSignatureTokenTruncation(t *testing.T) {
	// Arrange
	plaintext := "p"
	token := make([]byte, 32868+10)
	for i := range token {
		token[i] = byte(i)
	}
	truncated := token[:32868]
	mac := hmac.New(sha256.New, truncated)
	_, _ = mac.Write([]byte(plaintext))
	expected := mac.Sum(nil)

	// Act
	got := CalculateAccessDataSignature(&protocol.AccessData{}, token, plaintext)

	// Assert
	assert.Equal(t, expected, got)
}

func TestGenerateAccessDataFromHandshakeSignatures(t *testing.T) {
	// Arrange
	ad := &protocol.AccessData{}
	busID := uint64(0x11)
	nonce1 := uint64(1)
	nonce2 := uint64(2)
	t1 := []byte("t1")
	t2 := []byte("t2")
	hd := &protocol.CryptoHandshakeData{Type: protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_NONE, PublicKey: nil}

	now := time.Now().Unix()

	// Act
	GenerateAccessData(ad, busID, nonce1, nonce2, [][]byte{t1, t2}, hd)

	// Assert
	assert.Equal(t, protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256, ad.Algorithm)
	assert.Equal(t, nonce1, ad.Nonce1)
	assert.Equal(t, nonce2, ad.Nonce2)
	assert.Len(t, ad.Signature, 2)
	assert.GreaterOrEqual(t, ad.Timestamp, now-2)
	assert.LessOrEqual(t, ad.Timestamp, now+2)

	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)
	assert.Equal(t, CalculateAccessDataSignature(ad, t1, plaintext), ad.Signature[0])
	assert.Equal(t, CalculateAccessDataSignature(ad, t2, plaintext), ad.Signature[1])
}

func TestPackUnpackMessageRoundTrip(t *testing.T) {
	// Arrange
	ctx := impl.NewConnectionContext()
	payload := []byte("hello world")
	msgType := int32(123)
	source := uint64(0x22)

	// Act
	encoded, packed, err := PackMessage(ctx, payload, msgType, source, 0)
	assert.NoError(t, err)
	assert.NotNil(t, encoded)
	assert.NotNil(t, packed)

	decodedPacked, decodedPayload, err := UnpackMessage(ctx, encoded, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, decodedPacked)
	assert.NotNil(t, decodedPacked.Header)
	assert.Equal(t, msgType, decodedPacked.Header.Type)
	assert.Equal(t, source, decodedPacked.Header.SourceBusID)
	assert.Equal(t, payload, decodedPayload)
}

func TestPackMessagePayloadTooLarge(t *testing.T) {
	// Arrange
	ctx := impl.NewConnectionContext()
	payload := []byte("0123456789")

	// Act
	_, _, err := PackMessage(ctx, payload, 1, 1, 5)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, libatbus.EN_ATBUS_ERR_BUFF_LIMIT))
}

func TestUnpackMessageBodyTooLarge(t *testing.T) {
	// Arrange
	ctx := impl.NewConnectionContext()
	payload := []byte("0123456789")
	encoded, packed, err := PackMessage(ctx, payload, 1, 1, 0)
	assert.NoError(t, err)
	assert.NotNil(t, packed)
	assert.NotNil(t, packed.Header)

	// Act
	gotPacked, gotPayload, err := UnpackMessage(ctx, encoded, 1)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, libatbus.EN_ATBUS_ERR_BUFF_LIMIT))
	assert.NotNil(t, gotPacked)
	assert.Nil(t, gotPayload)
}

// ============================================================================
// Cross-language validation tests
// These tests use test data generated by C++ atbus_access_data_crosslang_generator.cpp
// to verify that Go and C++ implementations produce identical results.
// Test data is loaded from testdata/*.bytes binary files (matching C++ verification pattern).
// ============================================================================

// loadBinaryTestData loads binary data from the testdata directory.
func loadBinaryTestData(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read binary test data file: %s", path)
	return data
}

// TestCrossLangPlaintextNoPubkey verifies plaintext generation without public key
// matches the C++ implementation.
func TestCrossLangPlaintextNoPubkey(t *testing.T) {
	// Arrange - same parameters as C++ generator: plaintext_no_pubkey
	busID := uint64(0x12345678)            // 305419896
	timestamp := int64(1735689600)
	nonce1 := uint64(0x123456789ABCDEF0)   // 1311768467463790320
	nonce2 := uint64(0xFEDCBA9876543210)   // 18364758544493064720

	ad := &protocol.AccessData{
		Timestamp: timestamp,
		Nonce1:    nonce1,
		Nonce2:    nonce2,
	}
	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_NONE,
		PublicKey: nil,
	}

	// Expected format: "<timestamp>:<nonce1>-<nonce2>:<bus_id>"
	expected := "1735689600:1311768467463790320-18364758544493064720:305419896"

	// Act
	got := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Assert
	assert.Equal(t, expected, got, "Plaintext without pubkey should match C++ output")
}

// TestCrossLangPlaintextWithPubkey verifies plaintext generation with public key
// matches the C++ implementation (includes SHA256 of public key).
func TestCrossLangPlaintextWithPubkey(t *testing.T) {
	// Arrange - same parameters as C++ generator: plaintext_with_pubkey
	busID := uint64(0xABCDEF01)            // 2882400001
	timestamp := int64(1735689600)
	nonce1 := uint64(0xAAAABBBBCCCCDDDD)   // 12297848147757817309
	nonce2 := uint64(0x1111222233334444)   // 1229801703532086340

	// Public key: 00010203...1f (32 bytes)
	pubKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		pubKey[i] = byte(i)
	}

	ad := &protocol.AccessData{
		Timestamp: timestamp,
		Nonce1:    nonce1,
		Nonce2:    nonce2,
	}
	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_SECP256R1,
		PublicKey: pubKey,
	}

	// Act
	got := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Assert - verify format includes type and SHA256 hash
	assert.Contains(t, got, ":2:", "Plaintext should contain type indicator")
	assert.Contains(t, got, "630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd",
		"Plaintext should contain SHA256 of public key")
}

// TestCrossLangPlaintextCustomCommand verifies plaintext generation with custom command
// matches the C++ implementation (includes SHA256 of concatenated commands).
func TestCrossLangPlaintextCustomCommand(t *testing.T) {
	// Arrange - same parameters as C++ generator: plaintext_custom_command
	busID := uint64(0x87654321)            // 2271560481
	timestamp := int64(1735689600)
	nonce1 := uint64(0x5555666677778888)   // 6148933456521300104
	nonce2 := uint64(0x9999AAAABBBBCCCC)   // 11068065209510513868

	ad := &protocol.AccessData{
		Timestamp: timestamp,
		Nonce1:    nonce1,
		Nonce2:    nonce2,
	}

	cs := &protocol.CustomCommandData{
		Commands: []*protocol.CustomCommandArgv{
			{Arg: []byte("command1")},
			{Arg: []byte("arg2")},
			{Arg: []byte("data3")},
		},
	}

	// SHA256 of "command1arg2data3": 5f6b34d74c0c6c03c2c39f40d3dab18a027f4458f9ff7de47dd45e6cc5ef13e4
	expected := "1735689600:6148933456521300104-11068065209510513868:2271560481:5f6b34d74c0c6c03c2c39f40d3dab18a027f4458f9ff7de47dd45e6cc5ef13e4"

	// Act
	got := MakeAccessDataPlaintextFromCustomCommand(busID, ad, cs)

	// Assert
	assert.Equal(t, expected, got, "Plaintext with custom command should match C++ output")
}

// TestCrossLangSignatureSimpleToken reads the binary signature file and verifies
// that recalculating with the same parameters produces an identical result.
func TestCrossLangSignatureSimpleToken(t *testing.T) {
	// Load expected signature from binary file (same as C++ verify_signature_from_generated_files)
	expectedSig := loadBinaryTestData(t, "signature_simple_token.bytes")
	require.Len(t, expectedSig, 32, "HMAC-SHA256 signature should be 32 bytes")

	// Arrange - same parameters as C++ generator
	accessToken := []byte("secret_token_123")
	plaintext := "1735689600:1311768467294899695-18364758544106544929:305419896"

	ad := &protocol.AccessData{
		Algorithm: protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256,
		Timestamp: 1735689600,
		Nonce1:    0x1234567890ABCDEF,
		Nonce2:    0xFEDCBA0987654321,
	}

	// Act
	got := CalculateAccessDataSignature(ad, accessToken, plaintext)

	// Assert
	assert.Equal(t, expectedSig, got, "Signature should match binary file from C++ generator")
}

// TestCrossLangSignatureBinaryToken reads the binary signature file and verifies
// that recalculating with a binary token produces an identical result.
func TestCrossLangSignatureBinaryToken(t *testing.T) {
	// Load expected signature from binary file
	expectedSig := loadBinaryTestData(t, "signature_binary_token.bytes")
	require.Len(t, expectedSig, 32, "HMAC-SHA256 signature should be 32 bytes")

	// Arrange - same parameters as C++ generator
	// Binary token: (i * 7 + 13) & 0xFF for i in 0..31
	accessToken := make([]byte, 32)
	for i := 0; i < 32; i++ {
		accessToken[i] = byte((i*7 + 13) & 0xFF)
	}

	plaintext := "1735689600:12302652057474621457-2459565876494606489:2882400001"

	ad := &protocol.AccessData{
		Algorithm: protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256,
		Timestamp: 1735689600,
		Nonce1:    0xAABBCCDDEEFF0011,
		Nonce2:    0x2233445566778899,
	}

	// Act
	got := CalculateAccessDataSignature(ad, accessToken, plaintext)

	// Assert
	assert.Equal(t, expectedSig, got, "Signature with binary token should match binary file from C++ generator")
}

// TestCrossLangSignature256BitToken reads the binary signature file and verifies
// that recalculating with a 256-bit token produces an identical result.
func TestCrossLangSignature256BitToken(t *testing.T) {
	// Load expected signature from binary file
	expectedSig := loadBinaryTestData(t, "signature_256bit_token.bytes")
	require.Len(t, expectedSig, 32, "HMAC-SHA256 signature should be 32 bytes")

	// Arrange - same parameters as C++ generator
	accessToken := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
		0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA, 0x99, 0x88,
		0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00,
	}

	busID := uint64(0x99887766)

	ad := &protocol.AccessData{
		Algorithm: protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256,
		Timestamp: 1735689600,
		Nonce1:    0x1111111111111111,
		Nonce2:    0x2222222222222222,
	}

	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_NONE,
		PublicKey: nil,
	}

	// Generate plaintext using same method as C++
	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Act
	got := CalculateAccessDataSignature(ad, accessToken, plaintext)

	// Assert
	assert.Equal(t, expectedSig, got, "Signature with 256-bit token should match binary file from C++ generator")
}

// TestCrossLangFullAccessDataNoPubkey reads the serialized protobuf binary file,
// parses it, and verifies signatures by recalculating with the same parameters.
func TestCrossLangFullAccessDataNoPubkey(t *testing.T) {
	// Load serialized access_data from binary file
	serializedData := loadBinaryTestData(t, "full_access_data_no_pubkey.bytes")

	// Parse protobuf
	ad := &protocol.AccessData{}
	err := proto.Unmarshal(serializedData, ad)
	require.NoError(t, err, "Failed to parse access_data protobuf")

	// Verify basic fields
	assert.Equal(t, protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256, ad.Algorithm)
	assert.Equal(t, int64(1735689600), ad.Timestamp)
	assert.Equal(t, uint64(0xABCDEF0123456789), ad.Nonce1)
	assert.Equal(t, uint64(0x9876543210FEDCBA), ad.Nonce2)
	assert.Len(t, ad.Signature, 2)

	// Rebuild plaintext and verify signatures
	busID := uint64(0x12345678)
	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_NONE,
		PublicKey: nil,
	}

	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Verify first token signature
	token1 := []byte("token1")
	sig1 := CalculateAccessDataSignature(ad, token1, plaintext)
	assert.Equal(t, sig1, ad.Signature[0], "First signature should match recalculated value")

	// Verify second token signature
	token2 := []byte("token2")
	sig2 := CalculateAccessDataSignature(ad, token2, plaintext)
	assert.Equal(t, sig2, ad.Signature[1], "Second signature should match recalculated value")
}

// TestCrossLangFullAccessDataWithPubkey reads the serialized protobuf binary file,
// parses it, and verifies signatures including public key in plaintext.
func TestCrossLangFullAccessDataWithPubkey(t *testing.T) {
	// Load serialized access_data from binary file
	serializedData := loadBinaryTestData(t, "full_access_data_with_pubkey.bytes")

	// Parse protobuf
	ad := &protocol.AccessData{}
	err := proto.Unmarshal(serializedData, ad)
	require.NoError(t, err, "Failed to parse access_data protobuf")

	// Verify basic fields
	assert.Equal(t, protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256, ad.Algorithm)
	assert.Equal(t, int64(1735689600), ad.Timestamp)
	assert.Len(t, ad.Signature, 1)

	// Rebuild crypto_handshake_data (same as C++ generator)
	busID := uint64(0x87654321)
	pubKey := make([]byte, 65)
	pubKey[0] = 0x04 // Uncompressed point indicator
	for i := 1; i < 65; i++ {
		pubKey[i] = byte((i * 3) & 0xFF)
	}

	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_SECP256R1,
		PublicKey: pubKey,
	}

	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Verify token signature
	token := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	sig := CalculateAccessDataSignature(ad, token, plaintext)
	assert.Equal(t, sig, ad.Signature[0], "Signature with pubkey should match recalculated value")
}

// TestCrossLangFullAccessDataMultipleTokens reads the serialized protobuf binary file,
// parses it, and verifies all three signatures.
func TestCrossLangFullAccessDataMultipleTokens(t *testing.T) {
	// Load serialized access_data from binary file
	serializedData := loadBinaryTestData(t, "full_access_data_multiple_tokens.bytes")

	// Parse protobuf
	ad := &protocol.AccessData{}
	err := proto.Unmarshal(serializedData, ad)
	require.NoError(t, err, "Failed to parse access_data protobuf")

	// Verify basic fields
	assert.Equal(t, protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256, ad.Algorithm)
	assert.Len(t, ad.Signature, 3)

	// Rebuild plaintext
	busID := uint64(0xFEDCBA98)
	hd := &protocol.CryptoHandshakeData{
		Type:      protocol.ATBUS_CRYPTO_KEY_EXCHANGE_TYPE_ATBUS_CRYPTO_KEY_EXCHANGE_NONE,
		PublicKey: nil,
	}

	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)

	// Verify all three token signatures
	tokens := [][]byte{
		{'t', 'o', 'k', 'e', 'n', '_', 'a'},
		{'t', 'o', 'k', 'e', 'n', '_', 'b'},
		{'t', 'o', 'k', 'e', 'n', '_', 'c'},
	}

	for i, token := range tokens {
		sig := CalculateAccessDataSignature(ad, token, plaintext)
		assert.Equal(t, sig, ad.Signature[i], "Signature %d should match recalculated value", i+1)
	}
}

// Uncovered scenarios (by design):
// 1) Protobuf descriptor-driven body name lookup (C++ uses reflection); Go version uses a static map.
// 2) End-to-end node/connection dispatch handlers: the Go repo currently models Node/Connection as empty interfaces.
// 3) Access token verification and replay-window checks: those belong to higher-level node logic, not message_handle helpers.
