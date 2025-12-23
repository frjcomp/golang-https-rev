// Package protocol defines constants for the reverse shell protocol,
// including buffer sizes, timeouts, command names, and protocol markers.
package protocol

// Protocol constants used across the reverse shell system

const (
	// Buffer sizes
	BufferSize1MB = 1024 * 1024      // 1MB buffer for large file transfers
	MaxBufferSize = 10 * 1024 * 1024 // 10MB maximum accumulated buffer before reset
	ChunkSize     = 65536            // 64KB for file upload chunks

	// Protocol delimiters and markers
	EndOfOutputMarker = "<<<END_OF_OUTPUT>>>"
	DataPrefix        = "DATA "

	// Commands
	CmdPing        = "PING"
	CmdPong        = "PONG"
	CmdAuth        = "AUTH"        // Authentication handshake
	CmdAuthOk      = "AUTH_OK"     // Authentication successful
	CmdAuthFailed  = "AUTH_FAILED" // Authentication failed
	CmdExit        = "exit"
	CmdStartUpload = "START_UPLOAD"
	CmdUploadChunk = "UPLOAD_CHUNK"
	CmdEndUpload   = "END_UPLOAD"
	CmdDownload    = "DOWNLOAD"

	// PTY Mode Commands
	CmdPtyMode   = "PTY_MODE"   // Enter PTY shell mode
	CmdPtyData   = "PTY_DATA"   // PTY data stream
	CmdPtyResize = "PTY_RESIZE" // PTY window resize
	CmdPtyExit   = "PTY_EXIT"   // Exit PTY mode

	// Timeouts
	ReadTimeout     = 1          // second
	ResponseTimeout = 5          // seconds
	CommandTimeout  = 120        // seconds for shell command responses
	DownloadTimeout = 5000000000 // nanoseconds (very large for big files)
	PingInterval    = 30         // seconds
)
