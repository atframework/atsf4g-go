// Package libatbus_impl provides internal implementation details for libatbus.
// This file implements the connection context with encryption/decryption algorithm negotiation,
// compression algorithm negotiation, encryption/decryption flow, and pack/unpack flow.
package libatbus_impl

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	types "github.com/atframework/libatbus-go/types"
)

var _ types.ConnectionContext = (*ConnectionContext)(nil)

// CryptoAlgorithmType represents the encryption algorithm type.
// This corresponds to ATBUS_CRYPTO_ALGORITHM_TYPE in the protobuf definition.
type CryptoAlgorithmType int32

const (
	CryptoAlgorithmNone              CryptoAlgorithmType = 0
	CryptoAlgorithmXXTEA             CryptoAlgorithmType = 1
	CryptoAlgorithmAES128CBC         CryptoAlgorithmType = 11
	CryptoAlgorithmAES192CBC         CryptoAlgorithmType = 12
	CryptoAlgorithmAES256CBC         CryptoAlgorithmType = 13
	CryptoAlgorithmAES128GCM         CryptoAlgorithmType = 14
	CryptoAlgorithmAES192GCM         CryptoAlgorithmType = 15
	CryptoAlgorithmAES256GCM         CryptoAlgorithmType = 16
	CryptoAlgorithmChacha20          CryptoAlgorithmType = 31
	CryptoAlgorithmChacha20Poly1305  CryptoAlgorithmType = 32
	CryptoAlgorithmXChacha20Poly1305 CryptoAlgorithmType = 33
)

// String returns the string representation of the crypto algorithm type.
func (c CryptoAlgorithmType) String() string {
	switch c {
	case CryptoAlgorithmNone:
		return "NONE"
	case CryptoAlgorithmXXTEA:
		return "XXTEA"
	case CryptoAlgorithmAES128CBC:
		return "AES-128-CBC"
	case CryptoAlgorithmAES192CBC:
		return "AES-192-CBC"
	case CryptoAlgorithmAES256CBC:
		return "AES-256-CBC"
	case CryptoAlgorithmAES128GCM:
		return "AES-128-GCM"
	case CryptoAlgorithmAES192GCM:
		return "AES-192-GCM"
	case CryptoAlgorithmAES256GCM:
		return "AES-256-GCM"
	case CryptoAlgorithmChacha20:
		return "CHACHA20"
	case CryptoAlgorithmChacha20Poly1305:
		return "CHACHA20-POLY1305"
	case CryptoAlgorithmXChacha20Poly1305:
		return "XCHACHA20-POLY1305"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}

// KeySize returns the key size in bytes for the crypto algorithm.
func (c CryptoAlgorithmType) KeySize() int {
	switch c {
	case CryptoAlgorithmNone:
		return 0
	case CryptoAlgorithmXXTEA:
		return 16 // 128 bits
	case CryptoAlgorithmAES128CBC, CryptoAlgorithmAES128GCM:
		return 16 // 128 bits
	case CryptoAlgorithmAES192CBC, CryptoAlgorithmAES192GCM:
		return 24 // 192 bits
	case CryptoAlgorithmAES256CBC, CryptoAlgorithmAES256GCM:
		return 32 // 256 bits
	case CryptoAlgorithmChacha20, CryptoAlgorithmChacha20Poly1305, CryptoAlgorithmXChacha20Poly1305:
		return 32 // 256 bits
	default:
		return 0
	}
}

// IVSize returns the IV/nonce size in bytes for the crypto algorithm.
func (c CryptoAlgorithmType) IVSize() int {
	switch c {
	case CryptoAlgorithmNone:
		return 0
	case CryptoAlgorithmXXTEA:
		return 0
	case CryptoAlgorithmAES128CBC, CryptoAlgorithmAES192CBC, CryptoAlgorithmAES256CBC:
		return aes.BlockSize // 16 bytes
	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM:
		return 12 // standard GCM nonce size
	case CryptoAlgorithmChacha20:
		return 12
	case CryptoAlgorithmChacha20Poly1305:
		return chacha20poly1305.NonceSize // 12 bytes
	case CryptoAlgorithmXChacha20Poly1305:
		return chacha20poly1305.NonceSizeX // 24 bytes
	default:
		return 0
	}
}

// IsAEAD returns true if the algorithm is an AEAD cipher.
func (c CryptoAlgorithmType) IsAEAD() bool {
	switch c {
	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM,
		CryptoAlgorithmChacha20Poly1305, CryptoAlgorithmXChacha20Poly1305:
		return true
	default:
		return false
	}
}

// TagSize returns the authentication tag size for AEAD ciphers.
func (c CryptoAlgorithmType) TagSize() int {
	switch c {
	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM:
		return 16 // GCM tag size
	case CryptoAlgorithmChacha20Poly1305, CryptoAlgorithmXChacha20Poly1305:
		return chacha20poly1305.Overhead // 16 bytes
	default:
		return 0
	}
}

// KeyExchangeType represents the key exchange algorithm type.
// This corresponds to ATBUS_CRYPTO_KEY_EXCHANGE_TYPE in the protobuf definition.
type KeyExchangeType int32

const (
	KeyExchangeNone      KeyExchangeType = 0
	KeyExchangeX25519    KeyExchangeType = 1
	KeyExchangeSecp256r1 KeyExchangeType = 2
	KeyExchangeSecp384r1 KeyExchangeType = 3
	KeyExchangeSecp521r1 KeyExchangeType = 4
)

// String returns the string representation of the key exchange type.
func (k KeyExchangeType) String() string {
	switch k {
	case KeyExchangeNone:
		return "NONE"
	case KeyExchangeX25519:
		return "X25519"
	case KeyExchangeSecp256r1:
		return "SECP256R1"
	case KeyExchangeSecp384r1:
		return "SECP384R1"
	case KeyExchangeSecp521r1:
		return "SECP521R1"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", k)
	}
}

// Curve returns the corresponding ecdh.Curve for the key exchange type.
func (k KeyExchangeType) Curve() ecdh.Curve {
	switch k {
	case KeyExchangeX25519:
		return ecdh.X25519()
	case KeyExchangeSecp256r1:
		return ecdh.P256()
	case KeyExchangeSecp384r1:
		return ecdh.P384()
	case KeyExchangeSecp521r1:
		return ecdh.P521()
	default:
		return nil
	}
}

// KDFType represents the key derivation function type.
// This corresponds to ATBUS_CRYPTO_KDF_TYPE in the protobuf definition.
type KDFType int32

const (
	KDFTypeHKDFSha256 KDFType = 0
)

// String returns the string representation of the KDF type.
func (k KDFType) String() string {
	switch k {
	case KDFTypeHKDFSha256:
		return "HKDF-SHA256"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", k)
	}
}

// CompressionAlgorithmType represents the compression algorithm type.
// This corresponds to ATBUS_COMPRESSION_ALGORITHM_TYPE in the protobuf definition.
type CompressionAlgorithmType int32

const (
	CompressionNone   CompressionAlgorithmType = 0
	CompressionZstd   CompressionAlgorithmType = 100
	CompressionLZ4    CompressionAlgorithmType = 200
	CompressionSnappy CompressionAlgorithmType = 300
	CompressionZlib   CompressionAlgorithmType = 400
)

// String returns the string representation of the compression algorithm type.
func (c CompressionAlgorithmType) String() string {
	switch c {
	case CompressionNone:
		return "NONE"
	case CompressionZstd:
		return "ZSTD"
	case CompressionLZ4:
		return "LZ4"
	case CompressionSnappy:
		return "SNAPPY"
	case CompressionZlib:
		return "ZLIB"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}

// Error definitions for connection context.
var (
	ErrCryptoNotInitialized        = errors.New("crypto not initialized")
	ErrCryptoAlgorithmNotSupported = errors.New("crypto algorithm not supported")
	ErrCryptoInvalidKeySize        = errors.New("invalid crypto key size")
	ErrCryptoInvalidIVSize         = errors.New("invalid crypto iv/nonce size")
	ErrCryptoEncryptFailed         = errors.New("crypto encrypt failed")
	ErrCryptoDecryptFailed         = errors.New("crypto decrypt failed")
	ErrCryptoHandshakeFailed       = errors.New("crypto handshake failed")
	ErrCryptoKeyExchangeFailed     = errors.New("crypto key exchange failed")
	ErrCryptoKDFFailed             = errors.New("crypto kdf failed")
	ErrCompressionNotSupported     = errors.New("compression algorithm not supported")
	ErrCompressionFailed           = errors.New("compression failed")
	ErrDecompressionFailed         = errors.New("decompression failed")
	ErrPackFailed                  = errors.New("pack failed")
	ErrUnpackFailed                = errors.New("unpack failed")
	ErrInvalidData                 = errors.New("invalid data")
	ErrConnectionClosing           = errors.New("connection is closing")
)

// CryptoHandshakeData holds the data for crypto handshake.
type CryptoHandshakeData struct {
	Sequence    uint64
	KeyExchange KeyExchangeType
	KDFTypes    []KDFType
	Algorithms  []CryptoAlgorithmType
	PublicKey   []byte
	IVSize      uint32
	TagSize     uint32
}

// CryptoSession holds the crypto session state.
type CryptoSession struct {
	mu sync.RWMutex

	// Negotiated algorithm and parameters
	Algorithm   CryptoAlgorithmType
	KeyExchange KeyExchangeType
	KDFType     KDFType
	Key         []byte
	IV          []byte
	TagSize     uint32
	IVSize      uint32

	// ECDH key pair
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey

	// Cipher instances (cached for performance)
	aeadCipher  cipher.AEAD
	blockCipher cipher.Block

	// Nonce counter for AEAD modes
	nonceCounter uint64

	initialized bool
}

// NewCryptoSession creates a new crypto session.
func NewCryptoSession() *CryptoSession {
	return &CryptoSession{}
}

// IsInitialized returns true if the crypto session is initialized.
func (cs *CryptoSession) IsInitialized() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.initialized
}

// GenerateKeyPair generates a new ECDH key pair for the given key exchange type.
func (cs *CryptoSession) GenerateKeyPair(keyExchange KeyExchangeType) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	curve := keyExchange.Curve()
	if curve == nil {
		return fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, keyExchange.String())
	}

	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCryptoHandshakeFailed, err)
	}

	cs.privateKey = privateKey
	cs.publicKey = privateKey.PublicKey()
	cs.KeyExchange = keyExchange

	return nil
}

// GetPublicKey returns the public key bytes.
func (cs *CryptoSession) GetPublicKey() []byte {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if cs.publicKey == nil {
		return nil
	}
	return cs.publicKey.Bytes()
}

// ComputeSharedSecret computes the shared secret using the peer's public key.
func (cs *CryptoSession) ComputeSharedSecret(peerPublicKeyBytes []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.privateKey == nil {
		return nil, ErrCryptoNotInitialized
	}

	curve := cs.KeyExchange.Curve()
	if curve == nil {
		return nil, fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, cs.KeyExchange.String())
	}

	peerPublicKey, err := curve.NewPublicKey(peerPublicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCryptoKeyExchangeFailed, err)
	}

	sharedSecret, err := cs.privateKey.ECDH(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCryptoKeyExchangeFailed, err)
	}

	return sharedSecret, nil
}

// DeriveKey derives the encryption key and IV from the shared secret using HKDF.
func (cs *CryptoSession) DeriveKey(sharedSecret []byte, algorithm CryptoAlgorithmType, kdfType KDFType) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if kdfType != KDFTypeHKDFSha256 {
		return fmt.Errorf("%w: %s", ErrCryptoKDFFailed, kdfType.String())
	}

	keySize := algorithm.KeySize()
	ivSize := algorithm.IVSize()
	if keySize == 0 && algorithm != CryptoAlgorithmNone {
		return fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, algorithm.String())
	}

	// Derive key material using HKDF
	totalSize := keySize + ivSize
	if totalSize == 0 {
		// No encryption needed
		cs.Algorithm = algorithm
		cs.KDFType = kdfType
		cs.initialized = true
		return nil
	}

	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, nil)
	keyMaterial := make([]byte, totalSize)
	if _, err := io.ReadFull(hkdfReader, keyMaterial); err != nil {
		return fmt.Errorf("%w: %v", ErrCryptoKDFFailed, err)
	}

	cs.Key = keyMaterial[:keySize]
	if ivSize > 0 {
		cs.IV = keyMaterial[keySize:]
	}
	cs.Algorithm = algorithm
	cs.KDFType = kdfType
	cs.IVSize = uint32(ivSize)
	cs.TagSize = uint32(algorithm.TagSize())

	// Initialize cipher
	if err := cs.initCipher(); err != nil {
		return err
	}

	cs.initialized = true
	return nil
}

// SetKey directly sets the encryption key and IV.
func (cs *CryptoSession) SetKey(key, iv []byte, algorithm CryptoAlgorithmType) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	expectedKeySize := algorithm.KeySize()
	expectedIVSize := algorithm.IVSize()

	if algorithm == CryptoAlgorithmNone {
		cs.Algorithm = algorithm
		cs.initialized = true
		return nil
	}

	if len(key) != expectedKeySize {
		return fmt.Errorf("%w: expected %d, got %d", ErrCryptoInvalidKeySize, expectedKeySize, len(key))
	}

	if len(iv) != expectedIVSize && expectedIVSize > 0 {
		return fmt.Errorf("%w: expected %d, got %d", ErrCryptoInvalidIVSize, expectedIVSize, len(iv))
	}

	cs.Key = make([]byte, len(key))
	copy(cs.Key, key)
	if len(iv) > 0 {
		cs.IV = make([]byte, len(iv))
		copy(cs.IV, iv)
	}
	cs.Algorithm = algorithm
	cs.IVSize = uint32(expectedIVSize)
	cs.TagSize = uint32(algorithm.TagSize())

	if err := cs.initCipher(); err != nil {
		return err
	}

	cs.initialized = true
	return nil
}

// initCipher initializes the cipher based on the algorithm.
// Caller must hold the lock.
func (cs *CryptoSession) initCipher() error {
	switch cs.Algorithm {
	case CryptoAlgorithmNone:
		return nil

	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM:
		block, err := aes.NewCipher(cs.Key)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCryptoAlgorithmNotSupported, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCryptoAlgorithmNotSupported, err)
		}
		cs.aeadCipher = aead
		cs.blockCipher = block

	case CryptoAlgorithmAES128CBC, CryptoAlgorithmAES192CBC, CryptoAlgorithmAES256CBC:
		block, err := aes.NewCipher(cs.Key)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCryptoAlgorithmNotSupported, err)
		}
		cs.blockCipher = block

	case CryptoAlgorithmChacha20Poly1305:
		aead, err := chacha20poly1305.New(cs.Key)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCryptoAlgorithmNotSupported, err)
		}
		cs.aeadCipher = aead

	case CryptoAlgorithmXChacha20Poly1305:
		aead, err := chacha20poly1305.NewX(cs.Key)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCryptoAlgorithmNotSupported, err)
		}
		cs.aeadCipher = aead

	default:
		return fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, cs.Algorithm.String())
	}

	return nil
}

// Encrypt encrypts the plaintext data.
func (cs *CryptoSession) Encrypt(plaintext []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.initialized {
		return nil, ErrCryptoNotInitialized
	}

	if cs.Algorithm == CryptoAlgorithmNone {
		result := make([]byte, len(plaintext))
		copy(result, plaintext)
		return result, nil
	}

	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	switch cs.Algorithm {
	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM,
		CryptoAlgorithmChacha20Poly1305, CryptoAlgorithmXChacha20Poly1305:
		return cs.encryptAEAD(plaintext)

	case CryptoAlgorithmAES128CBC, CryptoAlgorithmAES192CBC, CryptoAlgorithmAES256CBC:
		return cs.encryptCBC(plaintext)

	default:
		return nil, fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, cs.Algorithm.String())
	}
}

// encryptAEAD encrypts using AEAD cipher.
// Caller must hold the lock.
func (cs *CryptoSession) encryptAEAD(plaintext []byte) ([]byte, error) {
	if cs.aeadCipher == nil {
		return nil, ErrCryptoNotInitialized
	}

	nonceSize := cs.aeadCipher.NonceSize()
	nonce := make([]byte, nonceSize)

	// Generate nonce: use counter + random for uniqueness
	counter := atomic.AddUint64(&cs.nonceCounter, 1)
	binary.LittleEndian.PutUint64(nonce, counter)
	if nonceSize > 8 {
		// Fill remaining bytes with random data
		if _, err := rand.Read(nonce[8:]); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCryptoEncryptFailed, err)
		}
	}

	// Encrypt: nonce || ciphertext || tag
	ciphertext := cs.aeadCipher.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	result := make([]byte, nonceSize+len(ciphertext))
	copy(result[:nonceSize], nonce)
	copy(result[nonceSize:], ciphertext)

	return result, nil
}

// encryptCBC encrypts using CBC mode with PKCS#7 padding.
// Caller must hold the lock.
func (cs *CryptoSession) encryptCBC(plaintext []byte) ([]byte, error) {
	if cs.blockCipher == nil {
		return nil, ErrCryptoNotInitialized
	}

	blockSize := cs.blockCipher.BlockSize()

	// Apply PKCS#7 padding
	padding := blockSize - (len(plaintext) % blockSize)
	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padding)
	}

	// Generate random IV
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCryptoEncryptFailed, err)
	}

	// Encrypt
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(cs.blockCipher, iv)
	mode.CryptBlocks(ciphertext, padded)

	// Prepend IV to ciphertext
	result := make([]byte, blockSize+len(ciphertext))
	copy(result[:blockSize], iv)
	copy(result[blockSize:], ciphertext)

	return result, nil
}

// Decrypt decrypts the ciphertext data.
func (cs *CryptoSession) Decrypt(ciphertext []byte) ([]byte, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if !cs.initialized {
		return nil, ErrCryptoNotInitialized
	}

	if cs.Algorithm == CryptoAlgorithmNone {
		result := make([]byte, len(ciphertext))
		copy(result, ciphertext)
		return result, nil
	}

	if len(ciphertext) == 0 {
		return []byte{}, nil
	}

	switch cs.Algorithm {
	case CryptoAlgorithmAES128GCM, CryptoAlgorithmAES192GCM, CryptoAlgorithmAES256GCM,
		CryptoAlgorithmChacha20Poly1305, CryptoAlgorithmXChacha20Poly1305:
		return cs.decryptAEAD(ciphertext)

	case CryptoAlgorithmAES128CBC, CryptoAlgorithmAES192CBC, CryptoAlgorithmAES256CBC:
		return cs.decryptCBC(ciphertext)

	default:
		return nil, fmt.Errorf("%w: %s", ErrCryptoAlgorithmNotSupported, cs.Algorithm.String())
	}
}

// decryptAEAD decrypts using AEAD cipher.
// Caller must hold the lock.
func (cs *CryptoSession) decryptAEAD(ciphertext []byte) ([]byte, error) {
	if cs.aeadCipher == nil {
		return nil, ErrCryptoNotInitialized
	}

	nonceSize := cs.aeadCipher.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrCryptoDecryptFailed)
	}

	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	plaintext, err := cs.aeadCipher.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCryptoDecryptFailed, err)
	}

	return plaintext, nil
}

// decryptCBC decrypts using CBC mode and removes PKCS#7 padding.
// Caller must hold the lock.
func (cs *CryptoSession) decryptCBC(ciphertext []byte) ([]byte, error) {
	if cs.blockCipher == nil {
		return nil, ErrCryptoNotInitialized
	}

	blockSize := cs.blockCipher.BlockSize()
	if len(ciphertext) < blockSize*2 {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrCryptoDecryptFailed)
	}

	if len(ciphertext)%blockSize != 0 {
		return nil, fmt.Errorf("%w: invalid ciphertext length", ErrCryptoDecryptFailed)
	}

	iv := ciphertext[:blockSize]
	encryptedData := ciphertext[blockSize:]

	// Decrypt
	plaintext := make([]byte, len(encryptedData))
	mode := cipher.NewCBCDecrypter(cs.blockCipher, iv)
	mode.CryptBlocks(plaintext, encryptedData)

	// Remove PKCS#7 padding
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("%w: invalid padding", ErrCryptoDecryptFailed)
	}

	padding := int(plaintext[len(plaintext)-1])
	if padding <= 0 || padding > blockSize {
		return nil, fmt.Errorf("%w: invalid padding value", ErrCryptoDecryptFailed)
	}

	// Verify padding
	for i := len(plaintext) - padding; i < len(plaintext); i++ {
		if plaintext[i] != byte(padding) {
			return nil, fmt.Errorf("%w: invalid padding bytes", ErrCryptoDecryptFailed)
		}
	}

	return plaintext[:len(plaintext)-padding], nil
}

// CompressionSession handles compression and decompression.
type CompressionSession struct {
	mu        sync.RWMutex
	Algorithm CompressionAlgorithmType
}

// NewCompressionSession creates a new compression session.
func NewCompressionSession() *CompressionSession {
	return &CompressionSession{
		Algorithm: CompressionNone,
	}
}

// SetAlgorithm sets the compression algorithm.
func (cs *CompressionSession) SetAlgorithm(algorithm CompressionAlgorithmType) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	switch algorithm {
	case CompressionNone, CompressionZlib:
		cs.Algorithm = algorithm
		return nil
	case CompressionZstd, CompressionLZ4, CompressionSnappy:
		// These require external libraries, mark as not supported for now
		// TODO: Add support with optional build tags
		return fmt.Errorf("%w: %s (external library required)", ErrCompressionNotSupported, algorithm.String())
	default:
		return fmt.Errorf("%w: %s", ErrCompressionNotSupported, algorithm.String())
	}
}

// GetAlgorithm returns the current compression algorithm.
func (cs *CompressionSession) GetAlgorithm() CompressionAlgorithmType {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.Algorithm
}

// Compress compresses the data using the configured algorithm.
func (cs *CompressionSession) Compress(data []byte) ([]byte, error) {
	cs.mu.RLock()
	algorithm := cs.Algorithm
	cs.mu.RUnlock()

	if len(data) == 0 {
		return []byte{}, nil
	}

	switch algorithm {
	case CompressionNone:
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil

	case CompressionZlib:
		return cs.compressZlib(data)

	default:
		return nil, fmt.Errorf("%w: %s", ErrCompressionNotSupported, algorithm.String())
	}
}

// compressZlib compresses data using zlib.
func (cs *CompressionSession) compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCompressionFailed, err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCompressionFailed, err)
	}
	return buf.Bytes(), nil
}

// Decompress decompresses the data using the configured algorithm.
func (cs *CompressionSession) Decompress(data []byte) ([]byte, error) {
	cs.mu.RLock()
	algorithm := cs.Algorithm
	cs.mu.RUnlock()

	if len(data) == 0 {
		return []byte{}, nil
	}

	switch algorithm {
	case CompressionNone:
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil

	case CompressionZlib:
		return cs.decompressZlib(data)

	default:
		return nil, fmt.Errorf("%w: %s", ErrCompressionNotSupported, algorithm.String())
	}
}

// decompressZlib decompresses data using zlib.
func (cs *CompressionSession) decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecompressionFailed, err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecompressionFailed, err)
	}

	return buf.Bytes(), nil
}

// NegotiateCompression negotiates the compression algorithm based on supported algorithms.
func NegotiateCompression(local, remote []CompressionAlgorithmType) CompressionAlgorithmType {
	// Priority order: ZSTD > LZ4 > SNAPPY > ZLIB > NONE
	priority := []CompressionAlgorithmType{
		CompressionZstd,
		CompressionLZ4,
		CompressionSnappy,
		CompressionZlib,
		CompressionNone,
	}

	localSet := make(map[CompressionAlgorithmType]bool)
	for _, alg := range local {
		localSet[alg] = true
	}

	remoteSet := make(map[CompressionAlgorithmType]bool)
	for _, alg := range remote {
		remoteSet[alg] = true
	}

	for _, alg := range priority {
		if localSet[alg] && remoteSet[alg] {
			return alg
		}
	}

	return CompressionNone
}

// NegotiateCryptoAlgorithm negotiates the crypto algorithm based on supported algorithms.
func NegotiateCryptoAlgorithm(local, remote []CryptoAlgorithmType) CryptoAlgorithmType {
	// Priority order: AEAD algorithms first, then CBC
	priority := []CryptoAlgorithmType{
		CryptoAlgorithmXChacha20Poly1305,
		CryptoAlgorithmChacha20Poly1305,
		CryptoAlgorithmAES256GCM,
		CryptoAlgorithmAES192GCM,
		CryptoAlgorithmAES128GCM,
		CryptoAlgorithmAES256CBC,
		CryptoAlgorithmAES192CBC,
		CryptoAlgorithmAES128CBC,
		CryptoAlgorithmNone,
	}

	localSet := make(map[CryptoAlgorithmType]bool)
	for _, alg := range local {
		localSet[alg] = true
	}

	remoteSet := make(map[CryptoAlgorithmType]bool)
	for _, alg := range remote {
		remoteSet[alg] = true
	}

	for _, alg := range priority {
		if localSet[alg] && remoteSet[alg] {
			return alg
		}
	}

	return CryptoAlgorithmNone
}

// NegotiateKeyExchange negotiates the key exchange algorithm.
func NegotiateKeyExchange(local, remote KeyExchangeType) KeyExchangeType {
	// Both sides must agree on the same key exchange type
	if local == remote {
		return local
	}
	return KeyExchangeNone
}

// NegotiateKDF negotiates the KDF type based on supported types.
func NegotiateKDF(local, remote []KDFType) KDFType {
	localSet := make(map[KDFType]bool)
	for _, kdf := range local {
		localSet[kdf] = true
	}

	for _, kdf := range remote {
		if localSet[kdf] {
			return kdf
		}
	}

	return KDFTypeHKDFSha256 // Default
}

// ConnectionContext manages the connection state including crypto and compression.
type ConnectionContext struct {
	mu sync.RWMutex

	// Crypto sessions
	readCrypto      *CryptoSession
	writeCrypto     *CryptoSession
	handshakeCrypto *CryptoSession

	// Compression session
	compression *CompressionSession

	// Connection state
	sequence      uint64
	closing       bool
	handshakeDone bool

	// Supported algorithms
	supportedCryptoAlgorithms      []CryptoAlgorithmType
	supportedCompressionAlgorithms []CompressionAlgorithmType
	supportedKeyExchange           KeyExchangeType
	supportedKDFTypes              []KDFType
}

// NewConnectionContext creates a new connection context with default settings.
func NewConnectionContext() *ConnectionContext {
	return &ConnectionContext{
		readCrypto:      NewCryptoSession(),
		writeCrypto:     NewCryptoSession(),
		handshakeCrypto: NewCryptoSession(),
		compression:     NewCompressionSession(),
		supportedCryptoAlgorithms: []CryptoAlgorithmType{
			CryptoAlgorithmAES256GCM,
			CryptoAlgorithmAES128GCM,
			CryptoAlgorithmChacha20Poly1305,
			CryptoAlgorithmXChacha20Poly1305,
			CryptoAlgorithmAES256CBC,
			CryptoAlgorithmAES128CBC,
			CryptoAlgorithmNone,
		},
		supportedCompressionAlgorithms: []CompressionAlgorithmType{
			CompressionZlib,
			CompressionNone,
		},
		supportedKeyExchange: KeyExchangeX25519,
		supportedKDFTypes: []KDFType{
			KDFTypeHKDFSha256,
		},
	}
}

// IsClosing returns true if the connection is closing.
func (cc *ConnectionContext) IsClosing() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.closing
}

// SetClosing sets the closing state.
func (cc *ConnectionContext) SetClosing(closing bool) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.closing = closing
}

// IsHandshakeDone returns true if the handshake is completed.
func (cc *ConnectionContext) IsHandshakeDone() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.handshakeDone
}

// GetNextSequence returns the next sequence number.
func (cc *ConnectionContext) GetNextSequence() uint64 {
	return atomic.AddUint64(&cc.sequence, 1)
}

// GetReadCrypto returns the read crypto session.
func (cc *ConnectionContext) GetReadCrypto() *CryptoSession {
	return cc.readCrypto
}

// GetWriteCrypto returns the write crypto session.
func (cc *ConnectionContext) GetWriteCrypto() *CryptoSession {
	return cc.writeCrypto
}

// GetCompression returns the compression session.
func (cc *ConnectionContext) GetCompression() *CompressionSession {
	return cc.compression
}

// SetSupportedCryptoAlgorithms sets the supported crypto algorithms.
func (cc *ConnectionContext) SetSupportedCryptoAlgorithms(algorithms []CryptoAlgorithmType) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.supportedCryptoAlgorithms = make([]CryptoAlgorithmType, len(algorithms))
	copy(cc.supportedCryptoAlgorithms, algorithms)
}

// GetSupportedCryptoAlgorithms returns the supported crypto algorithms.
func (cc *ConnectionContext) GetSupportedCryptoAlgorithms() []CryptoAlgorithmType {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	result := make([]CryptoAlgorithmType, len(cc.supportedCryptoAlgorithms))
	copy(result, cc.supportedCryptoAlgorithms)
	return result
}

// SetSupportedCompressionAlgorithms sets the supported compression algorithms.
func (cc *ConnectionContext) SetSupportedCompressionAlgorithms(algorithms []CompressionAlgorithmType) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.supportedCompressionAlgorithms = make([]CompressionAlgorithmType, len(algorithms))
	copy(cc.supportedCompressionAlgorithms, algorithms)
}

// GetSupportedCompressionAlgorithms returns the supported compression algorithms.
func (cc *ConnectionContext) GetSupportedCompressionAlgorithms() []CompressionAlgorithmType {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	result := make([]CompressionAlgorithmType, len(cc.supportedCompressionAlgorithms))
	copy(result, cc.supportedCompressionAlgorithms)
	return result
}

// SetSupportedKeyExchange sets the supported key exchange type.
func (cc *ConnectionContext) SetSupportedKeyExchange(keyExchange KeyExchangeType) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.supportedKeyExchange = keyExchange
}

// GetSupportedKeyExchange returns the supported key exchange type.
func (cc *ConnectionContext) GetSupportedKeyExchange() KeyExchangeType {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.supportedKeyExchange
}

// CreateHandshakeData creates the handshake data for initiating a handshake.
func (cc *ConnectionContext) CreateHandshakeData() (*CryptoHandshakeData, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closing {
		return nil, ErrConnectionClosing
	}

	// Generate key pair
	if err := cc.handshakeCrypto.GenerateKeyPair(cc.supportedKeyExchange); err != nil {
		return nil, err
	}

	return &CryptoHandshakeData{
		Sequence:    atomic.AddUint64(&cc.sequence, 1),
		KeyExchange: cc.supportedKeyExchange,
		KDFTypes:    cc.supportedKDFTypes,
		Algorithms:  cc.supportedCryptoAlgorithms,
		PublicKey:   cc.handshakeCrypto.GetPublicKey(),
		IVSize:      0, // Will be set after negotiation
		TagSize:     0, // Will be set after negotiation
	}, nil
}

// ProcessHandshakeData processes the received handshake data and completes the key exchange.
func (cc *ConnectionContext) ProcessHandshakeData(peerData *CryptoHandshakeData) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closing {
		return ErrConnectionClosing
	}

	// Negotiate key exchange
	keyExchange := NegotiateKeyExchange(cc.supportedKeyExchange, peerData.KeyExchange)
	if keyExchange == KeyExchangeNone && cc.supportedKeyExchange != KeyExchangeNone {
		return fmt.Errorf("%w: key exchange mismatch", ErrCryptoHandshakeFailed)
	}

	// Negotiate crypto algorithm
	algorithm := NegotiateCryptoAlgorithm(cc.supportedCryptoAlgorithms, peerData.Algorithms)

	// Negotiate KDF
	kdf := NegotiateKDF(cc.supportedKDFTypes, peerData.KDFTypes)

	// Generate key pair if not already done
	if cc.handshakeCrypto.privateKey == nil {
		if err := cc.handshakeCrypto.GenerateKeyPair(keyExchange); err != nil {
			return err
		}
	}

	// Compute shared secret
	sharedSecret, err := cc.handshakeCrypto.ComputeSharedSecret(peerData.PublicKey)
	if err != nil {
		return err
	}

	// Derive keys
	if err := cc.handshakeCrypto.DeriveKey(sharedSecret, algorithm, kdf); err != nil {
		return err
	}

	// Set up read and write crypto sessions with the same key
	if err := cc.readCrypto.SetKey(cc.handshakeCrypto.Key, cc.handshakeCrypto.IV, algorithm); err != nil {
		return err
	}
	if err := cc.writeCrypto.SetKey(cc.handshakeCrypto.Key, cc.handshakeCrypto.IV, algorithm); err != nil {
		return err
	}

	cc.handshakeDone = true
	return nil
}

// NegotiateCompressionWithPeer negotiates compression with peer's supported algorithms.
func (cc *ConnectionContext) NegotiateCompressionWithPeer(peerAlgorithms []CompressionAlgorithmType) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	algorithm := NegotiateCompression(cc.supportedCompressionAlgorithms, peerAlgorithms)
	return cc.compression.SetAlgorithm(algorithm)
}

// MessageHeader represents the header of a packed message.
//
// It is defined in `libatbus-go/types` and aliased here for convenience.
type MessageHeader = types.MessageHeader

// PackedMessage represents a packed message ready for transmission.
//
// It is defined in `libatbus-go/types` and aliased here for convenience.
type PackedMessage = types.PackedMessage

// Pack packs the message with optional compression and encryption.
func (cc *ConnectionContext) Pack(data []byte, msgType int32, sourceBusID uint64) (*PackedMessage, error) {
	cc.mu.RLock()
	if cc.closing {
		cc.mu.RUnlock()
		return nil, ErrConnectionClosing
	}
	cc.mu.RUnlock()

	originalSize := uint64(len(data))
	processedData := data

	// Step 1: Compress if needed
	compressionAlg := cc.compression.GetAlgorithm()
	if compressionAlg != CompressionNone && len(data) > 0 {
		compressed, err := cc.compression.Compress(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPackFailed, err)
		}
		// Only use compressed data if it's actually smaller
		if len(compressed) < len(data) {
			processedData = compressed
		} else {
			compressionAlg = CompressionNone
		}
	}

	// Step 2: Encrypt
	var cryptoIV []byte
	cryptoAlg := CryptoAlgorithmNone
	if cc.writeCrypto.IsInitialized() {
		encrypted, err := cc.writeCrypto.Encrypt(processedData)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPackFailed, err)
		}
		processedData = encrypted
		cryptoAlg = cc.writeCrypto.Algorithm
		// IV is embedded in the encrypted data for AEAD/CBC modes
	}

	header := &MessageHeader{
		Version:         3, // ATBUS_PROTOCOL_VERSION
		Type:            msgType,
		ResultCode:      0,
		Sequence:        cc.GetNextSequence(),
		SourceBusID:     sourceBusID,
		CryptoAlgorithm: int32(cryptoAlg),
		CryptoIV:        cryptoIV,
		CryptoAAD:       nil,
		CompressionType: int32(compressionAlg),
		OriginalSize:    originalSize,
		BodySize:        uint64(len(processedData)),
	}

	return &PackedMessage{
		Header: header,
		Body:   processedData,
	}, nil
}

// Unpack unpacks the message with optional decryption and decompression.
func (cc *ConnectionContext) Unpack(msg *PackedMessage) ([]byte, error) {
	cc.mu.RLock()
	if cc.closing {
		cc.mu.RUnlock()
		return nil, ErrConnectionClosing
	}
	cc.mu.RUnlock()

	if msg == nil || msg.Header == nil {
		return nil, fmt.Errorf("%w: nil message", ErrUnpackFailed)
	}

	processedData := msg.Body

	// Step 1: Decrypt if needed
	if msg.Header.CryptoAlgorithm != int32(CryptoAlgorithmNone) {
		if !cc.readCrypto.IsInitialized() {
			return nil, fmt.Errorf("%w: crypto not initialized", ErrUnpackFailed)
		}
		decrypted, err := cc.readCrypto.Decrypt(processedData)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnpackFailed, err)
		}
		processedData = decrypted
	}

	// Step 2: Decompress if needed
	if msg.Header.CompressionType != int32(CompressionNone) {
		// Temporarily set the decompression algorithm
		originalAlg := cc.compression.GetAlgorithm()
		if err := cc.compression.SetAlgorithm(CompressionAlgorithmType(msg.Header.CompressionType)); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnpackFailed, err)
		}
		decompressed, err := cc.compression.Decompress(processedData)
		// Restore original algorithm
		_ = cc.compression.SetAlgorithm(originalAlg)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnpackFailed, err)
		}
		processedData = decompressed
	}

	return processedData, nil
}

// EncodePackedMessage encodes a PackedMessage to bytes for transmission.
func EncodePackedMessage(msg *PackedMessage) ([]byte, error) {
	if msg == nil || msg.Header == nil {
		return nil, fmt.Errorf("%w: nil message", ErrPackFailed)
	}

	// Simple binary encoding format:
	// [4 bytes: header length][header bytes][body bytes]
	// Header format:
	// [4: version][4: type][4: result_code][8: sequence][8: source_bus_id]
	// [4: crypto_alg][4: iv_len][iv bytes][4: aad_len][aad bytes]
	// [4: compression_type][8: original_size][8: body_size]

	header := msg.Header
	ivLen := len(header.CryptoIV)
	aadLen := len(header.CryptoAAD)

	headerSize := 4 + 4 + 4 + 8 + 8 + // version, type, result_code, sequence, source_bus_id
		4 + 4 + ivLen + 4 + aadLen + // crypto fields
		4 + 8 + 8 // compression fields

	buf := make([]byte, 4+headerSize+len(msg.Body))
	offset := 0

	// Header length
	binary.LittleEndian.PutUint32(buf[offset:], uint32(headerSize))
	offset += 4

	// Version
	binary.LittleEndian.PutUint32(buf[offset:], uint32(header.Version))
	offset += 4

	// Type
	binary.LittleEndian.PutUint32(buf[offset:], uint32(header.Type))
	offset += 4

	// ResultCode
	binary.LittleEndian.PutUint32(buf[offset:], uint32(header.ResultCode))
	offset += 4

	// Sequence
	binary.LittleEndian.PutUint64(buf[offset:], header.Sequence)
	offset += 8

	// SourceBusID
	binary.LittleEndian.PutUint64(buf[offset:], header.SourceBusID)
	offset += 8

	// Crypto algorithm
	binary.LittleEndian.PutUint32(buf[offset:], uint32(header.CryptoAlgorithm))
	offset += 4

	// IV length and data
	binary.LittleEndian.PutUint32(buf[offset:], uint32(ivLen))
	offset += 4
	if ivLen > 0 {
		copy(buf[offset:], header.CryptoIV)
		offset += ivLen
	}

	// AAD length and data
	binary.LittleEndian.PutUint32(buf[offset:], uint32(aadLen))
	offset += 4
	if aadLen > 0 {
		copy(buf[offset:], header.CryptoAAD)
		offset += aadLen
	}

	// Compression type
	binary.LittleEndian.PutUint32(buf[offset:], uint32(header.CompressionType))
	offset += 4

	// Original size
	binary.LittleEndian.PutUint64(buf[offset:], header.OriginalSize)
	offset += 8

	// Body size
	binary.LittleEndian.PutUint64(buf[offset:], header.BodySize)
	offset += 8

	// Body
	copy(buf[offset:], msg.Body)

	return buf, nil
}

// DecodePackedMessage decodes bytes to a PackedMessage.
func DecodePackedMessage(data []byte) (*PackedMessage, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: data too short", ErrUnpackFailed)
	}

	offset := 0

	// Header length
	headerLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if len(data) < int(4+headerLen) {
		return nil, fmt.Errorf("%w: incomplete header", ErrUnpackFailed)
	}

	header := &MessageHeader{}

	// Version
	header.Version = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// Type
	header.Type = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// ResultCode
	header.ResultCode = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// Sequence
	header.Sequence = binary.LittleEndian.Uint64(data[offset:])
	offset += 8

	// SourceBusID
	header.SourceBusID = binary.LittleEndian.Uint64(data[offset:])
	offset += 8

	// Crypto algorithm
	header.CryptoAlgorithm = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// IV length and data
	ivLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if ivLen > 0 {
		if len(data) < offset+int(ivLen) {
			return nil, fmt.Errorf("%w: incomplete IV", ErrUnpackFailed)
		}
		header.CryptoIV = make([]byte, ivLen)
		copy(header.CryptoIV, data[offset:offset+int(ivLen)])
		offset += int(ivLen)
	}

	// AAD length and data
	aadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if aadLen > 0 {
		if len(data) < offset+int(aadLen) {
			return nil, fmt.Errorf("%w: incomplete AAD", ErrUnpackFailed)
		}
		header.CryptoAAD = make([]byte, aadLen)
		copy(header.CryptoAAD, data[offset:offset+int(aadLen)])
		offset += int(aadLen)
	}

	// Compression type
	header.CompressionType = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// Original size
	header.OriginalSize = binary.LittleEndian.Uint64(data[offset:])
	offset += 8

	// Body size
	header.BodySize = binary.LittleEndian.Uint64(data[offset:])
	offset += 8

	// Body
	if len(data) < offset+int(header.BodySize) {
		return nil, fmt.Errorf("%w: incomplete body", ErrUnpackFailed)
	}
	body := make([]byte, header.BodySize)
	copy(body, data[offset:offset+int(header.BodySize)])

	return &PackedMessage{
		Header: header,
		Body:   body,
	}, nil
}
