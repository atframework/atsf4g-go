package compression

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressionIsAlgorithmSupportedKnownValues(t *testing.T) {
	// Arrange: expected support flags for known algorithms
	cases := []struct {
		name      string
		algorithm Algorithm
		expected  bool
	}{
		{"ZSTD_Supported", AlgorithmZstd, true},
		{"LZ4_Supported", AlgorithmLz4, true},
		{"Snappy_Supported", AlgorithmSnappy, true},
		{"Zlib_Supported", AlgorithmZlib, true},
		{"None_NotSupported", AlgorithmNone, false},
		{"Unknown_NotSupported", Algorithm(999), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := IsAlgorithmSupported(tc.algorithm)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompressionGetSupportedAlgorithmsExpectedList(t *testing.T) {
	// Arrange
	// Act
	algs := GetSupportedAlgorithms()

	// Assert
	assert.Equal(t, []Algorithm{AlgorithmZstd, AlgorithmLz4, AlgorithmSnappy, AlgorithmZlib}, algs)
}

func TestCompressionGetAlgorithmNameKnownAndUnknown(t *testing.T) {
	// Arrange
	cases := []struct {
		algorithm Algorithm
		expected  string
	}{
		{AlgorithmZstd, "zstd"},
		{AlgorithmLz4, "lz4"},
		{AlgorithmSnappy, "snappy"},
		{AlgorithmZlib, "zlib"},
		{AlgorithmNone, "none"},
		{Algorithm(999), "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			// Act
			name := GetAlgorithmName(tc.algorithm)

			// Assert
			assert.Equal(t, tc.expected, name)
		})
	}
}

func TestCompressionMapCompressionLevelMappings(t *testing.T) {
	// Arrange
	cases := []struct {
		name     string
		alg      Algorithm
		level    Level
		expected MappedLevel
	}{
		{"Zstd_Default", AlgorithmZstd, LevelDefault, MappedLevel{Level: mapLevelZstd(LevelDefault), UseHighCompression: false}},
		{"Lz4_High", AlgorithmLz4, LevelHighRate, MappedLevel{Level: 9, UseHighCompression: true}},
		{"Snappy_Balanced", AlgorithmSnappy, LevelBalanced, MappedLevel{Level: 0, UseHighCompression: false}},
		{"Zlib_Balanced", AlgorithmZlib, LevelBalanced, MappedLevel{Level: 6, UseHighCompression: false}},
		{"None_Default", AlgorithmNone, LevelDefault, MappedLevel{Level: 0, UseHighCompression: false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			mapped := MapCompressionLevel(tc.alg, tc.level)

			// Assert
			assert.Equal(t, tc.expected, mapped)
		})
	}
}

func TestCompressionCompressNoneInvalidParam(t *testing.T) {
	// Arrange
	input := []byte("invalid")

	// Act
	_, code := Compress(AlgorithmNone, input, LevelDefault)

	// Assert
	assert.Equal(t, ErrorCodeInvalidParam, code)
}

func TestCompressionCompressWithRawLevelSnappyNotSupported(t *testing.T) {
	// Arrange
	input := []byte("snappy raw level")

	// Act
	_, code := CompressWithRawLevel(AlgorithmSnappy, input, 1)

	// Assert
	assert.Equal(t, ErrorCodeNotSupport, code)
}

func TestCompressionDecompressInvalidParam(t *testing.T) {
	// Arrange
	input := []byte("payload")

	// Act
	_, codeNone := Decompress(AlgorithmNone, input, 0)
	_, codeUnknown := Decompress(Algorithm(999), input, 0)
	_, codeLz4 := Decompress(AlgorithmLz4, input, 0)
	_, codeZlib := Decompress(AlgorithmZlib, input, 0)

	// Assert
	assert.Equal(t, ErrorCodeInvalidParam, codeNone)
	assert.Equal(t, ErrorCodeNotSupport, codeUnknown)
	assert.Equal(t, ErrorCodeInvalidParam, codeLz4)
	assert.Equal(t, ErrorCodeInvalidParam, codeZlib)
}

func TestCompressionRoundTripZstd(t *testing.T) {
	// Arrange: highly compressible payload
	input := bytes.Repeat([]byte("zstd-"), 2048)

	// Act
	compressed, code := Compress(AlgorithmZstd, input, LevelBalanced)
	require.Equal(t, ErrorCodeOk, code)
	decompressed, code := Decompress(AlgorithmZstd, compressed, len(input))
	require.Equal(t, ErrorCodeOk, code)

	// Assert
	assert.Equal(t, input, decompressed)
	assert.Less(t, len(compressed), len(input))
}

func TestCompressionRoundTripLz4(t *testing.T) {
	// Arrange: highly compressible payload
	input := bytes.Repeat([]byte("lz4-"), 2048)

	// Act
	compressed, code := Compress(AlgorithmLz4, input, LevelBalanced)
	require.Equal(t, ErrorCodeOk, code)
	decompressed, code := Decompress(AlgorithmLz4, compressed, len(input))
	require.Equal(t, ErrorCodeOk, code)

	// Assert
	assert.Equal(t, input, decompressed)
}

func TestCompressionRoundTripSnappy(t *testing.T) {
	// Arrange: highly compressible payload
	input := bytes.Repeat([]byte("snappy-"), 2048)

	// Act
	compressed, code := Compress(AlgorithmSnappy, input, LevelBalanced)
	require.Equal(t, ErrorCodeOk, code)
	decompressed, code := Decompress(AlgorithmSnappy, compressed, 0)
	require.Equal(t, ErrorCodeOk, code)

	// Assert
	assert.Equal(t, input, decompressed)
}

func TestCompressionRoundTripZlib(t *testing.T) {
	// Arrange: highly compressible payload
	input := bytes.Repeat([]byte("zlib-"), 2048)

	// Act
	compressed, code := Compress(AlgorithmZlib, input, LevelBalanced)
	require.Equal(t, ErrorCodeOk, code)
	decompressed, code := Decompress(AlgorithmZlib, compressed, len(input))
	require.Equal(t, ErrorCodeOk, code)

	// Assert
	assert.Equal(t, input, decompressed)
	assert.Less(t, len(compressed), len(input))
}

// Uncovered scenarios:
// - Decompression with corrupted payload per algorithm: omitted to keep tests concise.
// - Raw-level compression for zstd/lz4: covered by adapter mapping tests but not executed here.
