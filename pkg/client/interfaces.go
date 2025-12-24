package client

// ReverseClientInterface defines the interface for a reverse shell client that connects to a listener.
// It handles the full lifecycle of connecting to and communicating with a reverse shell listener.
type ReverseClientInterface interface {
	// Connect establishes a connection to the listener.
	// Returns an error if the connection fails.
	Connect() error

	// HandleCommands enters the command processing loop.
	// Blocks until the connection is closed or an error occurs.
	HandleCommands() error

	// Close gracefully closes the connection to the listener.
	Close() error

	// IsConnected returns whether the client is currently connected.
	IsConnected() bool
}

// CommandExecutor defines the interface for executing shell commands on the remote system.
type CommandExecutor interface {
	// ExecuteCommand runs a command and returns the output.
	ExecuteCommand(command string) string
}

// FileTransfer defines the interface for file operations (upload/download).
type FileTransfer interface {
	// Upload sends a file from local to remote.
	Upload(localPath, remotePath string) error

	// Download retrieves a file from remote to local.
	Download(remotePath, localPath string) error
}

// ClientStats defines the interface for retrieving client statistics.
type ClientStats interface {
	// GetConnectionAttempts returns the number of connection attempts made.
	GetConnectionAttempts() int

	// GetLastError returns the last error encountered.
	GetLastError() error
}
