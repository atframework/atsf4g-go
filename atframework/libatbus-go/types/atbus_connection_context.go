package libatbus_types

// MessageHeader represents the header of a packed message.
//
// NOTE: These fields mirror the subset used by the Go implementation under
// `libatbus-go/impl`. We intentionally keep algorithm fields as int32 to avoid
// importing implementation-specific enums into the shared `types` package.
type MessageHeader struct {
	Version         int32
	Type            int32
	ResultCode      int32
	Sequence        uint64
	SourceBusID     uint64
	CryptoAlgorithm int32
	CryptoIV        []byte
	CryptoAAD       []byte
	CompressionType int32
	OriginalSize    uint64
	BodySize        uint64
}

// PackedMessage represents a packed message ready for transmission.
type PackedMessage struct {
	Header *MessageHeader
	Body   []byte
}

// ConnectionContext abstracts the per-connection state and the pack/unpack flow.
//
// This interface is implemented by `libatbus-go/impl.ConnectionContext`.
// Keep it minimal and stable: only add methods that are required by callers.
type ConnectionContext interface {
	IsClosing() bool
	SetClosing(closing bool)

	IsHandshakeDone() bool

	// GetNextSequence returns the next sequence number.
	GetNextSequence() uint64

	// Pack packs payload data with optional compression/encryption.
	Pack(data []byte, msgType int32, sourceBusID uint64) (*PackedMessage, error)
	// Unpack unpacks payload data with optional decryption/decompression.
	Unpack(msg *PackedMessage) ([]byte, error)
}
