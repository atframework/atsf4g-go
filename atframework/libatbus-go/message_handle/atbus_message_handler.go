package libatbus_message_handle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	libatbus "github.com/atframework/libatbus-go"
	impl "github.com/atframework/libatbus-go/impl"
	protocol "github.com/atframework/libatbus-go/protocol"
	types "github.com/atframework/libatbus-go/types"
)

// This file is a Go port of the core helpers in atframework/libatbus
// `atbus_message_handler.cpp`.
//
// Scope note: only the following functions are fully implemented here:
//   - unpack_message
//   - pack_message
//   - get_body_name
//   - generate_access_data
//   - make_access_data_plaintext
//   - calculate_access_data_signature
//
// Other message dispatch / recv handlers are declared as interfaces/types only.

// ---- pack/unpack helpers ----

// UnpackMessage decodes a packed message and returns the decoded PackedMessage
// plus the decrypted/decompressed payload.
//
// maxBodySize follows the C++ signature: it bounds the encoded body size.
// If maxBodySize <= 0, the limit is not checked.
func UnpackMessage(connCtx types.ConnectionContext, data []byte, maxBodySize int) (*types.PackedMessage, []byte, error) {
	if connCtx == nil {
		return nil, nil, libatbus.EN_ATBUS_ERR_PARAMS
	}

	packed, err := impl.DecodePackedMessage(data)
	if err != nil {
		return nil, nil, err
	}

	if maxBodySize > 0 && packed != nil && packed.Header != nil {
		if packed.Header.BodySize > uint64(maxBodySize) {
			return packed, nil, libatbus.EN_ATBUS_ERR_BUFF_LIMIT
		}
	}

	payload, err := connCtx.Unpack(packed)
	if err != nil {
		return packed, nil, err
	}

	return packed, payload, nil
}

// PackMessage packs payload data and encodes it into the wire format.
//
// maxBodySize bounds the raw (pre-pack) payload size.
// If maxBodySize <= 0, the limit is not checked.
func PackMessage(connCtx types.ConnectionContext, payload []byte, msgType int32, sourceBusID uint64, maxBodySize int) ([]byte, *types.PackedMessage, error) {
	if connCtx == nil {
		return nil, nil, libatbus.EN_ATBUS_ERR_PARAMS
	}
	if maxBodySize > 0 && len(payload) > maxBodySize {
		return nil, nil, libatbus.EN_ATBUS_ERR_BUFF_LIMIT
	}

	packed, err := connCtx.Pack(payload, msgType, sourceBusID)
	if err != nil {
		return nil, nil, err
	}

	encoded, err := impl.EncodePackedMessage(packed)
	if err != nil {
		return nil, nil, err
	}

	return encoded, packed, nil
}

// ---- message body name helpers ----

// messageBodyFullNames maps oneof case IDs to their protobuf full names.
// Initialized from protobuf reflection at package load time.
var messageBodyFullNames map[int]string

func init() {
	// Build the body names map from protobuf reflection to avoid hardcoding.
	messageBodyFullNames = buildMessageBodyFullNames()
}

// buildMessageBodyFullNames uses protobuf reflection to build the mapping
// from oneof case IDs to their full field names.
func buildMessageBodyFullNames() map[int]string {
	result := make(map[int]string)

	// Get the message descriptor for MessageBody
	msg := &protocol.MessageBody{}
	md := msg.ProtoReflect().Descriptor()

	// Find the "message_type" oneof
	oneofs := md.Oneofs()
	for i := 0; i < oneofs.Len(); i++ {
		oneof := oneofs.Get(i)
		if oneof.Name() == "message_type" {
			// Iterate over all fields in this oneof
			fields := oneof.Fields()
			for j := 0; j < fields.Len(); j++ {
				field := fields.Get(j)
				fieldNum := int(field.Number())
				fullName := string(field.FullName())
				result[fieldNum] = fullName
			}
			break
		}
	}

	return result
}

// GetBodyName returns the protobuf full name of the oneof field for message_body.
//
// This mirrors C++ `message_handler::get_body_name()` which uses the protobuf
// descriptor's `FieldDescriptor::full_name()`.
func GetBodyName(bodyCase int) string {
	if n, ok := messageBodyFullNames[bodyCase]; ok && n != "" {
		return n
	}
	return "Unknown"
}

// ---- access_data helpers ----

// GenerateAccessData fills protocol.AccessData and appends one signature entry per token.
//
// It mirrors the C++ overload that takes crypto_handshake_data.
func GenerateAccessData(ad *protocol.AccessData, busID, nonce1, nonce2 uint64, accessTokens [][]byte, hd *protocol.CryptoHandshakeData) {
	GenerateAccessDataWithTimestamp(ad, busID, nonce1, nonce2, accessTokens, hd, time.Now().Unix())
}

// GenerateAccessDataWithTimestamp is like GenerateAccessData but allows specifying a fixed timestamp.
// This is primarily useful for cross-language compatibility testing.
func GenerateAccessDataWithTimestamp(ad *protocol.AccessData, busID, nonce1, nonce2 uint64, accessTokens [][]byte, hd *protocol.CryptoHandshakeData, timestamp int64) {
	if ad == nil {
		return
	}
	ad.Algorithm = protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256
	ad.Timestamp = timestamp
	ad.Nonce1 = nonce1
	ad.Nonce2 = nonce2

	plaintext := MakeAccessDataPlaintextFromHandshake(busID, ad, hd)
	ad.Signature = make([][]byte, 0, len(accessTokens))
	for _, token := range accessTokens {
		ad.Signature = append(ad.Signature, CalculateAccessDataSignature(ad, token, plaintext))
	}
}

// GenerateAccessDataForCustomCommand mirrors the C++ overload that takes custom_command_data.
func GenerateAccessDataForCustomCommand(ad *protocol.AccessData, busID, nonce1, nonce2 uint64, accessTokens [][]byte, cs *protocol.CustomCommandData) {
	if ad == nil {
		return
	}
	ad.Algorithm = protocol.ATBUS_ACCESS_DATA_ALGORITHM_TYPE_ATBUS_ACCESS_DATA_ALGORITHM_HMAC_SHA256
	ad.Timestamp = time.Now().Unix()
	ad.Nonce1 = nonce1
	ad.Nonce2 = nonce2

	plaintext := MakeAccessDataPlaintextFromCustomCommand(busID, ad, cs)
	ad.Signature = make([][]byte, 0, len(accessTokens))
	for _, token := range accessTokens {
		ad.Signature = append(ad.Signature, CalculateAccessDataSignature(ad, token, plaintext))
	}
}

// MakeAccessDataPlaintextFromHandshake builds the signed plaintext string.
//
// C++ rules:
//   - If public_key is empty:
//     "<timestamp>:<nonce1>-<nonce2>:<bus_id>"
//   - Else:
//     "<timestamp>:<nonce1>-<nonce2>:<bus_id>:<type>:<sha256_hex(public_key)>"
func MakeAccessDataPlaintextFromHandshake(busID uint64, ad *protocol.AccessData, hd *protocol.CryptoHandshakeData) string {
	if hd == nil || len(hd.GetPublicKey()) == 0 {
		return fmt.Sprintf("%d:%d-%d:%d", ad.GetTimestamp(), ad.GetNonce1(), ad.GetNonce2(), busID)
	}

	h := sha256.Sum256(hd.GetPublicKey())
	tailHash := hex.EncodeToString(h[:])
	return fmt.Sprintf("%d:%d-%d:%d:%d:%s", ad.GetTimestamp(), ad.GetNonce1(), ad.GetNonce2(), busID, int32(hd.GetType()), tailHash)
}

// MakeAccessDataPlaintextFromCustomCommand builds the signed plaintext string.
//
// C++ rules:
//
//	"<timestamp>:<nonce1>-<nonce2>:<bus_id>:<sha256_hex(concat(commands.arg))>"
func MakeAccessDataPlaintextFromCustomCommand(busID uint64, ad *protocol.AccessData, cs *protocol.CustomCommandData) string {
	// Concatenate all command args.
	total := 0
	commands := cs.GetCommands()
	for _, item := range commands {
		total += len(item.GetArg())
	}

	buf := make([]byte, 0, total)
	for _, item := range commands {
		buf = append(buf, item.GetArg()...)
	}

	h := sha256.Sum256(buf)
	return fmt.Sprintf("%d:%d-%d:%d:%s", ad.GetTimestamp(), ad.GetNonce1(), ad.GetNonce2(), busID, hex.EncodeToString(h[:]))
}

// CalculateAccessDataSignature computes the signature for the given plaintext.
//
// It mirrors C++:
//
//	signature = HMAC-SHA256(plaintext, access_token)
//
// and truncates access_token length to 32868 bytes.
func CalculateAccessDataSignature(_ *protocol.AccessData, accessToken []byte, plaintext string) []byte {
	token := accessToken
	if len(token) > 32868 {
		token = token[:32868]
	}

	mac := hmac.New(sha256.New, token)
	_, _ = mac.Write([]byte(plaintext))
	return mac.Sum(nil)
}

// ---- Message wrapper ----

// MessageBodyType is the Go equivalent of C++ message_body_type (the oneof case).
// It aliases protocol.MessageBody_EnMessageTypeID for convenience.
type MessageBodyType = protocol.MessageBody_EnMessageTypeID

// MessageBodyType constants using generated enum values.
const (
	MessageBodyTypeUnknown          = protocol.MessageBody_EnMessageTypeID_NONE
	MessageBodyTypeCustomCommandReq = protocol.MessageBody_EnMessageTypeID_CustomCommandReq
	MessageBodyTypeCustomCommandRsp = protocol.MessageBody_EnMessageTypeID_CustomCommandRsp
	MessageBodyTypeDataTransformReq = protocol.MessageBody_EnMessageTypeID_DataTransformReq
	MessageBodyTypeDataTransformRsp = protocol.MessageBody_EnMessageTypeID_DataTransformRsp
	MessageBodyTypeNodeSyncReq      = protocol.MessageBody_EnMessageTypeID_NodeSyncReq
	MessageBodyTypeNodeSyncRsp      = protocol.MessageBody_EnMessageTypeID_NodeSyncRsp
	MessageBodyTypeNodeRegisterReq  = protocol.MessageBody_EnMessageTypeID_NodeRegisterReq
	MessageBodyTypeNodeRegisterRsp  = protocol.MessageBody_EnMessageTypeID_NodeRegisterRsp
	MessageBodyTypeNodeConnectSync  = protocol.MessageBody_EnMessageTypeID_NodeConnectSync
	MessageBodyTypeNodePingReq      = protocol.MessageBody_EnMessageTypeID_NodePingReq
	MessageBodyTypeNodePongRsp      = protocol.MessageBody_EnMessageTypeID_NodePongRsp
)

// Message is the Go equivalent of C++ atbus::message.
//
// It wraps the protobuf-generated protocol.MessageHead and protocol.MessageBody,
// and provides accessor methods similar to the C++ class. The Go version does not
// use protobuf Arena or inplace caching; it holds the structs directly.
type Message struct {
	head *protocol.MessageHead
	body *protocol.MessageBody

	// unpackError stores any error encountered during unpacking.
	unpackError string
}

// NewMessage creates a new empty Message.
func NewMessage() *Message {
	return &Message{
		head: &protocol.MessageHead{},
		body: &protocol.MessageBody{},
	}
}

// MutableHead returns a mutable reference to the message head.
func (m *Message) MutableHead() *protocol.MessageHead {
	if m.head == nil {
		m.head = &protocol.MessageHead{}
	}
	return m.head
}

// MutableBody returns a mutable reference to the message body.
func (m *Message) MutableBody() *protocol.MessageBody {
	if m.body == nil {
		m.body = &protocol.MessageBody{}
	}
	return m.body
}

// GetHead returns the message head (may be nil).
func (m *Message) GetHead() *protocol.MessageHead {
	if m == nil {
		return nil
	}
	return m.head
}

// GetBody returns the message body (may be nil).
func (m *Message) GetBody() *protocol.MessageBody {
	if m == nil {
		return nil
	}
	return m.body
}

// Head returns the message head, creating an empty one if nil.
func (m *Message) Head() *protocol.MessageHead {
	return m.MutableHead()
}

// Body returns the message body, creating an empty one if nil.
func (m *Message) Body() *protocol.MessageBody {
	return m.MutableBody()
}

// GetBodyType returns the body oneof case based on the set message type.
// It uses the generated GetMessageTypeOneofCase() method.
func (m *Message) GetBodyType() MessageBodyType {
	if m == nil || m.body == nil {
		return MessageBodyTypeUnknown
	}
	return m.body.GetMessageTypeOneofCase()
}

// GetUnpackErrorMessage returns any unpack error message stored in the message.
func (m *Message) GetUnpackErrorMessage() string {
	if m == nil {
		return ""
	}
	return m.unpackError
}

// SetUnpackError sets the unpack error message.
func (m *Message) SetUnpackError(err string) {
	if m != nil {
		m.unpackError = err
	}
}

// ---- callback declarations (stubs) ----

// HandlerFn matches the C++ handler signature shape.
type HandlerFn func(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int

// Receiver declares message receive handlers.
// Implementations can be provided by higher-level packages.
type Receiver interface {
	OnRecvDataTransferReq(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvDataTransferRsp(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvCustomCmdReq(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvCustomCmdRsp(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodeSyncReq(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodeSyncRsp(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodeRegReq(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodeRegRsp(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodeConnSyn(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodePing(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
	OnRecvNodePong(n types.Node, conn types.Connection, msg *Message, status int, errcode int) int
}
