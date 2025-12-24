package protocol

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
	"testing"
)

func TestGetBuffer(t *testing.T) {
	// Get a buffer
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("GetBuffer returned nil")
	}

	// Buffer should be empty
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got length %d", buf.Len())
	}

	// Should be able to write to it
	n, err := buf.WriteString("test data")
	if err != nil {
		t.Fatalf("failed to write to buffer: %v", err)
	}
	if n != 9 {
		t.Errorf("expected 9 bytes written, got %d", n)
	}
}

func TestPutBuffer(t *testing.T) {
	// Get and populate a buffer
	buf := GetBuffer()
	buf.WriteString("test data")
	if buf.Len() == 0 {
		t.Fatal("buffer should have data")
	}

	// Put it back
	PutBuffer(buf)

	// Get a buffer again - it should be reset
	buf2 := GetBuffer()
	if buf2.Len() != 0 {
		t.Errorf("expected reset buffer, got length %d", buf2.Len())
	}
}

func TestPutBufferNil(t *testing.T) {
	// Should handle nil gracefully
	PutBuffer(nil)
	// No panic expected
}

func TestBufferPoolReuse(t *testing.T) {
	// Get first buffer
	buf1 := GetBuffer()
	buf1Ptr := buf1

	// Put it back
	PutBuffer(buf1)

	// Get next buffer - might be the same pointer
	buf2 := GetBuffer()

	// Check if it's reused (implementation detail, but good to verify)
	if buf2 == buf1Ptr {
		t.Log("✓ Buffer pool reuses buffers (expected)")
	}

	PutBuffer(buf2)
}

func TestGetGzipWriter(t *testing.T) {
	// Create a target buffer
	target := &bytes.Buffer{}

	// Get a gzip writer
	gw := GetGzipWriter(target)
	if gw == nil {
		t.Fatal("GetGzipWriter returned nil")
	}

	// Should be able to write through it
	if _, err := gw.Write([]byte("test data")); err != nil {
		t.Fatalf("failed to write to gzip writer: %v", err)
	}

	// Must close to flush
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	// Target should now contain compressed data
	if target.Len() == 0 {
		t.Error("target buffer should contain compressed data")
	}
}

func TestPutGzipWriter(t *testing.T) {
	// Get and use a gzip writer
	target := &bytes.Buffer{}
	gw := GetGzipWriter(target)
	gw.Write([]byte("test data"))
	gw.Close()

	// Put it back
	PutGzipWriter(gw)

	// Get another writer - might be the same pointer
	target2 := &bytes.Buffer{}
	gw2 := GetGzipWriter(target2)

	// Should be able to use it for compression
	gw2.Write([]byte("more test data"))
	gw2.Close()

	if target2.Len() == 0 {
		t.Error("second writer should produce compressed data")
	}
}

func TestPutGzipWriterNil(t *testing.T) {
	// Should handle nil gracefully
	PutGzipWriter(nil)
	// No panic expected
}

func TestBufferPoolConcurrency(t *testing.T) {
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track errors
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Get and use multiple buffers
			for j := 0; j < 10; j++ {
				buf := GetBuffer()
				if buf == nil {
					errChan <- ErrBufferNil()
					return
				}

				buf.WriteString("test")
				PutBuffer(buf)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestGzipWriterPoolConcurrency(t *testing.T) {
	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Get and use multiple gzip writers
			for j := 0; j < 5; j++ {
				target := &bytes.Buffer{}
				gw := GetGzipWriter(target)
				if gw == nil {
					errChan <- ErrGzipWriterNil()
					return
				}

				if _, err := gw.Write([]byte("test data")); err != nil {
					errChan <- err
					return
				}

				if err := gw.Close(); err != nil {
					errChan <- err
					return
				}

				PutGzipWriter(gw)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent gzip writer error: %v", err)
	}
}

// Helper error functions for tests
func ErrBufferNil() error {
	return io.ErrUnexpectedEOF // Using standard error as placeholder
}

func ErrGzipWriterNil() error {
	return io.ErrUnexpectedEOF // Using standard error as placeholder
}

func TestBufferMultipleResets(t *testing.T) {
	// Get a buffer and write to it multiple times
	buf := GetBuffer()

	for i := 0; i < 3; i++ {
		buf.WriteString("data")
		if buf.Len() == 0 {
			t.Errorf("iteration %d: buffer should have data", i)
		}
		PutBuffer(buf)

		// Get fresh buffer for next iteration
		buf = GetBuffer()
		if buf.Len() != 0 {
			t.Errorf("iteration %d: buffer should be reset", i)
		}
	}
}

func TestGzipWriterReset(t *testing.T) {
	// First use
	buf1 := &bytes.Buffer{}
	gw := GetGzipWriter(buf1)
	gw.Write([]byte("first"))
	gw.Close()

	size1 := buf1.Len()

	// Reset for second use
	buf2 := &bytes.Buffer{}
	gw = GetGzipWriter(buf2) // Reset to new target
	gw.Write([]byte("second"))
	gw.Close()

	size2 := buf2.Len()

	// Both should have compressed data (sizes may differ due to content)
	if size1 == 0 {
		t.Error("first compression should produce data")
	}
	if size2 == 0 {
		t.Error("second compression should produce data")
	}

	// Verify data can be decompressed
	gr, err := gzip.NewReader(buf2)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed := &bytes.Buffer{}
	if _, err := io.Copy(decompressed, gr); err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	if decompressed.String() != "second" {
		t.Errorf("expected 'second', got '%s'", decompressed.String())
	}
}
