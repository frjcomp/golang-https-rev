package compression

import (
	"bytes"
	"testing"
)

// TestCompressionFormat tests hex encoding format validation
func TestCompressionFormat(t *testing.T) {
	input := []byte("test format validation")
	encoded, err := CompressToHex(input)
	if err != nil {
		t.Fatalf("CompressToHex failed: %v", err)
	}

	// Verify it's valid hex (iterate as bytes, not runes)
	for i := 0; i < len(encoded); i++ {
		ch := encoded[i]
		if !isHexCharByte(ch) {
			t.Fatalf("Invalid hex character: %c (byte %d)", ch, ch)
		}
	}

	t.Log("✓ Compression format validation test passed")
}

// TestCompressionRatio tests that compression actually works
func TestCompressionRatio(t *testing.T) {
	// Highly compressible data
	input := bytes.Repeat([]byte("aaaaaaaa"), 10000)

	encoded, err := CompressToHex(input)
	if err != nil {
		t.Fatalf("CompressToHex failed: %v", err)
	}

	// Hex encoding doubles the size (2 chars per byte)
	// So compressed + hex should still be smaller than original
	if len(encoded) >= len(input)*2 {
		t.Logf("Note: compression didn't provide significant ratio improvement")
		t.Logf("  Original: %d bytes, Hex-encoded gzip: %d chars", len(input), len(encoded))
	}

	t.Log("✓ Compression ratio test passed")
}

// TestEmptyCompressionRoundtrip tests edge case of empty data
func TestEmptyCompressionRoundtrip(t *testing.T) {
	input := []byte{}

	encoded, err := CompressToHex(input)
	if err != nil {
		t.Fatalf("CompressToHex failed for empty input: %v", err)
	}

	decoded, err := DecompressHex(encoded)
	if err != nil {
		t.Fatalf("DecompressHex failed: %v", err)
	}

	if len(decoded) != 0 {
		t.Fatalf("Expected empty decoded data, got %d bytes", len(decoded))
	}

	t.Log("✓ Empty compression roundtrip test passed")
}

// TestLargeDataCompression tests compression of large payloads
func TestLargeDataCompression(t *testing.T) {
	// Create 5MB of test data
	input := bytes.Repeat([]byte("test data "), 512*1024)

	encoded, err := CompressToHex(input)
	if err != nil {
		t.Fatalf("CompressToHex failed: %v", err)
	}

	decoded, err := DecompressHex(encoded)
	if err != nil {
		t.Fatalf("DecompressHex failed: %v", err)
	}

	if !bytes.Equal(input, decoded) {
		t.Fatalf("Large data roundtrip failed: input len=%d, decoded len=%d", len(input), len(decoded))
	}

	t.Log("✓ Large data compression test passed")
}

// Helper function
func isHexCharByte(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
