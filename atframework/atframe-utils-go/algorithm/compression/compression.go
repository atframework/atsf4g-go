package compression

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Algorithm represents the compression algorithm type.
type Algorithm uint32

const (
	AlgorithmNone   Algorithm = 0
	AlgorithmZstd   Algorithm = 100
	AlgorithmLz4    Algorithm = 200
	AlgorithmSnappy Algorithm = 300
	AlgorithmZlib   Algorithm = 400
)

// Level represents unified compression levels.
type Level int32

const (
	LevelDefault  Level = 0
	LevelStorage  Level = 100
	LevelFast     Level = 200
	LevelLowCPU   Level = 300
	LevelBalanced Level = 400
	LevelHighRate Level = 500
	LevelMaxRate  Level = 600
)

// ErrorCode matches C++ compression error codes.
type ErrorCode int

const (
	ErrorCodeOk           ErrorCode = 0
	ErrorCodeInvalidParam ErrorCode = -1
	ErrorCodeNotSupport   ErrorCode = -2
	ErrorCodeBufferSmall  ErrorCode = -3
	ErrorCodeOperation    ErrorCode = -4
	ErrorCodeDisabled     ErrorCode = -5
)

// MappedLevel holds algorithm-specific parameters.
type MappedLevel struct {
	Level              int
	UseHighCompression bool
}

// IsAlgorithmSupported reports whether an algorithm is supported.
func IsAlgorithmSupported(alg Algorithm) bool {
	switch alg {
	case AlgorithmZstd, AlgorithmLz4, AlgorithmSnappy, AlgorithmZlib:
		return true
	case AlgorithmNone:
		return false
	default:
		return false
	}
}

// GetSupportedAlgorithms returns all supported algorithms.
func GetSupportedAlgorithms() []Algorithm {
	return []Algorithm{
		AlgorithmZstd,
		AlgorithmLz4,
		AlgorithmSnappy,
		AlgorithmZlib,
	}
}

// GetAlgorithmName returns the string name for an algorithm.
func GetAlgorithmName(alg Algorithm) string {
	switch alg {
	case AlgorithmZstd:
		return "zstd"
	case AlgorithmLz4:
		return "lz4"
	case AlgorithmSnappy:
		return "snappy"
	case AlgorithmZlib:
		return "zlib"
	case AlgorithmNone:
		return "none"
	default:
		return "unknown"
	}
}

// MapCompressionLevel maps unified compression level to algorithm-specific settings.
func MapCompressionLevel(alg Algorithm, level Level) MappedLevel {
	switch alg {
	case AlgorithmZstd:
		return MappedLevel{Level: mapLevelZstd(level), UseHighCompression: false}
	case AlgorithmLz4:
		return mapLevelLz4(level)
	case AlgorithmSnappy:
		return MappedLevel{Level: 0, UseHighCompression: false}
	case AlgorithmZlib:
		return MappedLevel{Level: mapLevelZlib(level), UseHighCompression: false}
	case AlgorithmNone:
		fallthrough
	default:
		return MappedLevel{Level: 0, UseHighCompression: false}
	}
}

// Compress compresses input data using the given algorithm and level.
func Compress(alg Algorithm, input []byte, level Level) ([]byte, ErrorCode) {
	return compressInternal(alg, input, false, 0, level)
}

// CompressWithRawLevel compresses input data using algorithm-specific raw level.
func CompressWithRawLevel(alg Algorithm, input []byte, rawLevel int) ([]byte, ErrorCode) {
	return compressInternal(alg, input, true, rawLevel, LevelDefault)
}

// Decompress decompresses input data using the given algorithm.
// originalSize is the expected size of the decompressed data; 0 means auto-detect when supported.
func Decompress(alg Algorithm, input []byte, originalSize int) ([]byte, ErrorCode) {
	if len(input) == 0 {
		return []byte{}, ErrorCodeOk
	}

	switch alg {
	case AlgorithmZstd:
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		defer decoder.Close()
		output, err := decoder.DecodeAll(input, nil)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		if originalSize > 0 && len(output) != originalSize {
			return nil, ErrorCodeOperation
		}
		return output, ErrorCodeOk

	case AlgorithmLz4:
		if originalSize <= 0 {
			return nil, ErrorCodeInvalidParam
		}
		output := make([]byte, originalSize)
		n, err := lz4.UncompressBlock(input, output)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		if n != originalSize {
			return nil, ErrorCodeOperation
		}
		return output[:n], ErrorCodeOk

	case AlgorithmSnappy:
		if originalSize == 0 {
			decodedLen, err := snappy.DecodedLen(input)
			if err != nil {
				return nil, ErrorCodeInvalidParam
			}
			originalSize = decodedLen
		}
		output, err := snappy.Decode(nil, input)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		if originalSize > 0 && len(output) != originalSize {
			return nil, ErrorCodeOperation
		}
		return output, ErrorCodeOk

	case AlgorithmZlib:
		if originalSize <= 0 {
			return nil, ErrorCodeInvalidParam
		}
		reader, err := zlib.NewReader(bytes.NewReader(input))
		if err != nil {
			return nil, ErrorCodeOperation
		}
		defer reader.Close()
		output, err := io.ReadAll(reader)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		if len(output) != originalSize {
			return nil, ErrorCodeOperation
		}
		return output, ErrorCodeOk

	case AlgorithmNone:
		return nil, ErrorCodeInvalidParam
	default:
		return nil, ErrorCodeNotSupport
	}
}

func compressInternal(alg Algorithm, input []byte, useRawLevel bool, rawLevel int, level Level) ([]byte, ErrorCode) {
	if len(input) == 0 {
		return []byte{}, ErrorCodeOk
	}

	switch alg {
	case AlgorithmZstd:
		zstdLevel := mapLevelZstd(level)
		if useRawLevel {
			zstdLevel = rawLevel
		}
		encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(zstdLevel)))
		if err != nil {
			return nil, ErrorCodeOperation
		}
		defer encoder.Close()
		output := encoder.EncodeAll(input, nil)
		return output, ErrorCodeOk

	case AlgorithmLz4:
		bound := lz4.CompressBlockBound(len(input))
		if bound <= 0 {
			return nil, ErrorCodeOperation
		}
		output := make([]byte, bound)
		var n int
		var err error

		if useRawLevel {
			if rawLevel <= 0 {
				n, err = lz4.CompressBlock(input, output, nil)
			} else if rawLevel < 3 {
				n, err = lz4.CompressBlock(input, output, nil)
			} else {
				clamped := clampInt(rawLevel, 3, 12)
				n, err = lz4.CompressBlockHC(input, output, lz4.CompressionLevel(clamped), nil, nil)
			}
		} else {
			mapped := mapLevelLz4(level)
			if mapped.UseHighCompression {
				clamped := clampInt(mapped.Level, 3, 12)
				n, err = lz4.CompressBlockHC(input, output, lz4.CompressionLevel(clamped), nil, nil)
			} else {
				n, err = lz4.CompressBlock(input, output, nil)
			}
		}

		if err != nil || n <= 0 {
			return nil, ErrorCodeOperation
		}
		return output[:n], ErrorCodeOk

	case AlgorithmSnappy:
		if useRawLevel {
			return nil, ErrorCodeNotSupport
		}
		return snappy.Encode(nil, input), ErrorCodeOk

	case AlgorithmZlib:
		zlibLevel := mapLevelZlib(level)
		if useRawLevel {
			zlibLevel = rawLevel
		}
		var buf bytes.Buffer
		writer, err := zlib.NewWriterLevel(&buf, zlibLevel)
		if err != nil {
			return nil, ErrorCodeOperation
		}
		if _, err := writer.Write(input); err != nil {
			_ = writer.Close()
			return nil, ErrorCodeOperation
		}
		if err := writer.Close(); err != nil {
			return nil, ErrorCodeOperation
		}
		return buf.Bytes(), ErrorCodeOk

	case AlgorithmNone:
		return nil, ErrorCodeInvalidParam
	default:
		return nil, ErrorCodeNotSupport
	}
}

func mapLevelLz4(level Level) MappedLevel {
	switch level {
	case LevelDefault:
		return MappedLevel{Level: 0, UseHighCompression: false}
	case LevelStorage:
		return MappedLevel{Level: 1, UseHighCompression: false}
	case LevelFast:
		return MappedLevel{Level: 1, UseHighCompression: false}
	case LevelLowCPU:
		return MappedLevel{Level: 3, UseHighCompression: false}
	case LevelBalanced:
		return MappedLevel{Level: 6, UseHighCompression: false}
	case LevelHighRate:
		return MappedLevel{Level: 9, UseHighCompression: true}
	case LevelMaxRate:
		return MappedLevel{Level: 12, UseHighCompression: true}
	default:
		return MappedLevel{Level: 0, UseHighCompression: false}
	}
}

func mapLevelZstd(level Level) int {
	switch level {
	case LevelDefault:
		return int(zstd.SpeedDefault)
	case LevelStorage:
		return -1
	case LevelFast:
		return -1
	case LevelLowCPU:
		return 1
	case LevelBalanced:
		return int(zstd.SpeedDefault)
	case LevelHighRate:
		return 6
	case LevelMaxRate:
		return 12
	default:
		return int(zstd.SpeedDefault)
	}
}

func mapLevelZlib(level Level) int {
	switch level {
	case LevelDefault:
		return flate.DefaultCompression
	case LevelStorage:
		return 1
	case LevelFast:
		return 1
	case LevelLowCPU:
		return 3
	case LevelBalanced:
		return 6
	case LevelHighRate:
		return 9
	case LevelMaxRate:
		return 9
	default:
		return 6
	}
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
