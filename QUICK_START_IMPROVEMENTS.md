# GOTS Quick-Start Improvement Guide

This document provides concrete code examples for implementing the most critical improvements.

## Quick Wins (Do These First)

### 1. Add Graceful Shutdown (30 minutes)

**Current State**: No way to shutdown cleanly

**Example Implementation**:

```go
// pkg/server/listener.go - Add context support
type Listener struct {
    port              string
    networkInterface  string
    tlsConfig         *tls.Config
    sharedSecret      string
    ctx               context.Context  // ADD THIS
    cancel            context.CancelFunc  // ADD THIS
    clientConnections map[string]chan string
    clientResponses   map[string]chan string
    clientPausePing   map[string]chan bool
    clientPtyMode     map[string]bool
    clientPtyData     map[string]chan []byte
    mutex             sync.Mutex
    wg                sync.WaitGroup  // ADD THIS - track goroutines
}

// NewListener - Add context
func NewListener(ctx context.Context, port, networkInterface string, tlsConfig *tls.Config, sharedSecret string) *Listener {
    ctx, cancel := context.WithCancel(ctx)
    return &Listener{
        port:              port,
        networkInterface:  networkInterface,
        tlsConfig:         tlsConfig,
        sharedSecret:      sharedSecret,
        ctx:               ctx,  // ADD THIS
        cancel:            cancel,  // ADD THIS
        clientConnections: make(map[string]chan string),
        clientResponses:   make(map[string]chan string),
        clientPausePing:   make(map[string]chan bool),
        clientPtyMode:     make(map[string]bool),
        clientPtyData:     make(map[string]chan []byte),
    }
}

// Start - Add WaitGroup tracking
func (l *Listener) Start() (net.Listener, error) {
    address := fmt.Sprintf("%s:%s", l.networkInterface, l.port)
    listener, err := tls.Listen("tcp", address, l.tlsConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create TLS listener: %w", err)
    }

    l.wg.Add(1)  // Track acceptConnections goroutine
    go func() {
        defer l.wg.Done()
        l.acceptConnections(listener)
    }()
    
    return listener, nil
}

// acceptConnections - Add context checking
func (l *Listener) acceptConnections(listener net.Listener) {
    for {
        select {
        case <-l.ctx.Done():  // Check for shutdown
            return
        default:
        }
        
        conn, err := listener.Accept()
        if err != nil {
            if errors.Is(err, net.ErrClosed) {
                return
            }
            log.Printf("Error accepting connection: %v", err)
            continue
        }
        
        l.wg.Add(1)  // Track client handler
        go func() {
            defer l.wg.Done()
            l.handleClient(conn)
        }()
    }
}

// Shutdown - Add graceful shutdown method
func (l *Listener) Shutdown(ctx context.Context) error {
    log.Printf("Initiating graceful shutdown...")
    l.cancel()  // Signal all goroutines to stop
    
    // Get all clients and close their connections
    l.mutex.Lock()
    clients := make([]string, 0, len(l.clientConnections))
    for addr := range l.clientConnections {
        clients = append(clients, addr)
    }
    l.mutex.Unlock()
    
    for _, addr := range clients {
        if err := l.SendCommand(addr, protocol.CmdExit); err != nil {
            log.Printf("Error sending exit to %s: %v", addr, err)
        }
    }
    
    // Wait for all goroutines to finish or timeout
    done := make(chan struct{})
    go func() {
        l.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        log.Printf("Graceful shutdown completed")
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// cmd/gotsl/main.go - Add signal handling
func main() {
    // ... existing setup ...
    
    listener := server.NewListener(context.Background(), port, networkInterface, tlsConfig, secret)
    netListener, err := listener.Start()
    if err != nil {
        return fmt.Errorf("failed to start listener: %w", err)
    }
    
    // Add signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Run interactive shell in goroutine
    shellDone := make(chan struct{})
    go func() {
        interactiveShell(listener)
        close(shellDone)
    }()
    
    // Wait for signal or shell exit
    select {
    case <-sigChan:
        log.Printf("Received shutdown signal")
    case <-shellDone:
        log.Printf("Interactive shell exited")
    }
    
    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    if err := listener.Shutdown(shutdownCtx); err != nil {
        log.Printf("Shutdown error: %v", err)
    }
    
    netListener.Close()
}
```

### 2. Implement Structured Logging (45 minutes)

**Current State**: Mixed fmt.Printf and log.Printf

**Example Implementation**:

```go
// pkg/logging/logger.go - NEW FILE
package logging

import (
    "log/slog"
    "os"
)

type Logger struct {
    *slog.Logger
}

func NewLogger(debug bool) *Logger {
    level := slog.LevelInfo
    if debug {
        level = slog.LevelDebug
    }
    
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    })
    
    return &Logger{
        Logger: slog.New(handler),
    }
}

// Convenience methods
func (l *Logger) Security(msg string, args ...any) {
    l.LogAttrs(nil, slog.LevelWarn, msg, append(args, slog.String("category", "security"))...)
}

func (l *Logger) Audit(msg string, action string, details map[string]any) {
    attrs := []any{
        slog.String("event", "audit"),
        slog.String("action", action),
    }
    for k, v := range details {
        attrs = append(attrs, slog.Any(k, v))
    }
    l.LogAttrs(nil, slog.LevelInfo, msg, attrs...)
}

// pkg/server/listener.go - Replace logging
func (l *Listener) handleClient(conn net.Conn) {
    clientAddr := conn.RemoteAddr().String()
    l.logger.Info("Client connected", slog.String("client", clientAddr))  // NEW
    
    // OLD: log.Printf("\n[+] New client connected: %s", clientAddr)
    
    if l.sharedSecret != "" {
        // Authentication failed
        l.logger.Security(  // NEW - audit log
            "Authentication failed",
            slog.String("client", clientAddr),
            slog.String("reason", "invalid_secret"),
        )
    }
}

// cmd/gotsl/main.go - Use structured logging
func runListener(args []string, useSharedSecret bool, logger *logging.Logger) error {
    logger.Info("Starting listener", 
        slog.String("port", port),
        slog.String("interface", networkInterface),
    )
    
    // OLD: log.Println("Generating self-signed certificate...")
    // NEW:
    logger.Info("Generating self-signed certificate")
}
```

### 3. Fix Path Traversal Vulnerabilities (45 minutes)

**Current State**: No validation of file paths

**Example Implementation**:

```go
// pkg/security/validation.go - NEW FILE
package security

import (
    "fmt"
    "path/filepath"
    "strings"
)

// ValidatePath ensures the path is within the allowed base directory
func ValidatePath(basePath, requestedPath string) (string, error) {
    // Reject absolute paths
    if filepath.IsAbs(requestedPath) {
        return "", fmt.Errorf("absolute paths not allowed")
    }
    
    // Clean path to remove ".." and "./"
    cleanPath := filepath.Clean(requestedPath)
    
    // Double-check there are no ".." components
    if strings.Contains(cleanPath, "..") {
        return "", fmt.Errorf("parent directory traversal not allowed")
    }
    
    // Construct full path
    fullPath := filepath.Join(basePath, cleanPath)
    
    // Verify the resolved path is still within basePath
    absBase, err := filepath.Abs(basePath)
    if err != nil {
        return "", fmt.Errorf("failed to resolve base path: %w", err)
    }
    
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        return "", fmt.Errorf("failed to resolve path: %w", err)
    }
    
    // Check that absPath starts with absBase/
    if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) &&
       absPath != absBase {
        return "", fmt.Errorf("path escapes base directory")
    }
    
    return absPath, nil
}

// pkg/client/command_handlers.go - Use validation
func (rc *ReverseClient) handleDownloadCommand(command string) error {
    parts := strings.SplitN(command, " ", 2)
    if len(parts) != 2 {
        rc.writer.WriteString("Invalid download command\n" + protocol.EndOfOutputMarker + "\n")
        rc.writer.Flush()
        return fmt.Errorf("invalid download command: %s", command)
    }

    requestedPath := parts[1]
    
    // VALIDATE PATH - NEW
    filePath, err := security.ValidatePath(rc.allowedDir, requestedPath)
    if err != nil {
        rc.writer.WriteString(fmt.Sprintf("Access denied: %v\n", err) + protocol.EndOfOutputMarker + "\n")
        rc.writer.Flush()
        return err
    }
    
    data, err := os.ReadFile(filePath)  // Now safe
    if err != nil {
        rc.writer.WriteString(fmt.Sprintf("Error reading file: %v\n", err) + protocol.EndOfOutputMarker + "\n")
        rc.writer.Flush()
        return fmt.Errorf("failed to read file: %w", err)
    }
    // ... rest of function
}
```

### 4. Fix Race Conditions with sync.RWMutex (1 hour)

**Current State**: Single mutex, multiple concurrent accesses

**Example Implementation**:

```go
// pkg/server/listener.go - Replace sync.Mutex with sync.RWMutex
type Listener struct {
    port              string
    networkInterface  string
    tlsConfig         *tls.Config
    sharedSecret      string
    ctx               context.Context
    cancel            context.CancelFunc
    clientConnections map[string]chan string
    clientResponses   map[string]chan string
    clientPausePing   map[string]chan bool
    clientPtyMode     map[string]bool
    clientPtyData     map[string]chan []byte
    mutex             sync.RWMutex  // Changed from sync.Mutex
    wg                sync.WaitGroup
}

// GetClients - Use read lock
func (l *Listener) GetClients() []string {
    l.mutex.RLock()  // Changed to RLock
    defer l.mutex.RUnlock()

    clients := make([]string, 0, len(l.clientConnections))
    for addr := range l.clientConnections {
        clients = append(clients, addr)
    }
    return clients
}

// SendCommand - Use write lock where needed
func (l *Listener) SendCommand(clientAddr, cmd string) error {
    l.mutex.RLock()  // Read lock for map access
    cmdChan, exists := l.clientConnections[clientAddr]
    pauseChan, pauseExists := l.clientPausePing[clientAddr]
    l.mutex.RUnlock()

    if !exists {
        return fmt.Errorf("client %s not found", clientAddr)
    }

    // Send without lock
    if pauseExists {
        select {
        case <-pauseChan:
        default:
        }
        select {
        case pauseChan <- true:
        default:
        }
    }

    select {
    case cmdChan <- cmd:
        return nil
    case <-l.ctx.Done():
        return l.ctx.Err()
    case <-time.After(5 * time.Second):
        return fmt.Errorf("command send timeout for client %s", clientAddr)
    }
}

// handleClient - Proper cleanup
func (l *Listener) handleClient(conn net.Conn) {
    clientAddr := conn.RemoteAddr().String()
    
    defer func() {
        conn.Close()
        l.mutex.Lock()  // Write lock for map cleanup
        delete(l.clientConnections, clientAddr)
        delete(l.clientResponses, clientAddr)
        delete(l.clientPausePing, clientAddr)
        if ptyDataChan, exists := l.clientPtyData[clientAddr]; exists {
            close(ptyDataChan)
            delete(l.clientPtyData, clientAddr)
        }
        delete(l.clientPtyMode, clientAddr)
        l.mutex.Unlock()
    }()
    
    // ... rest of function
}
```

### 5. Implement Buffer Pools for Memory Management (1 hour)

**Current State**: Large buffers allocated and discarded frequently

**Example Implementation**:

```go
// pkg/protocol/buffer_pool.go - NEW FILE
package protocol

import (
    "bytes"
    "sync"
)

type BufferPool struct {
    pool sync.Pool
    size int
}

func NewBufferPool(size int) *BufferPool {
    return &BufferPool{
        pool: sync.Pool{
            New: func() any {
                return bytes.NewBuffer(make([]byte, 0, size))
            },
        },
        size: size,
    }
}

func (bp *BufferPool) Get() *bytes.Buffer {
    buf := bp.pool.Get().(*bytes.Buffer)
    buf.Reset()
    return buf
}

func (bp *BufferPool) Put(buf *bytes.Buffer) {
    // Only return buffers of reasonable size
    if buf.Cap() <= bp.size*2 {
        bp.pool.Put(buf)
    }
}

// pkg/client/reverse.go - Use buffer pool
type ReverseClient struct {
    // ... existing fields ...
    bufferPool *protocol.BufferPool  // ADD THIS
}

func NewReverseClient(target, sharedSecret, certFingerprint string) *ReverseClient {
    return &ReverseClient{
        target:          target,
        sharedSecret:    sharedSecret,
        certFingerprint: certFingerprint,
        bufferPool:      protocol.NewBufferPool(protocol.BufferSize1MB),  // ADD THIS
    }
}

// HandleCommands - Use buffer pool
func (rc *ReverseClient) HandleCommands() error {
    cmdBuffer := rc.bufferPool.Get()  // Get from pool
    defer rc.bufferPool.Put(cmdBuffer)  // Return to pool
    
    for {
        // ... existing code ...
        
        // Before resetting
        if cmdBuffer.Len() > protocol.MaxBufferSize {
            cmdBuffer.Reset()
            // Don't return oversized buffers to pool
            cmdBuffer = rc.bufferPool.Get()
        }
    }
}
```

---

## Testing These Changes

```bash
# Test for race conditions
go test -race ./...

# Run with benchmarks
go test -bench=. -benchmem ./...

# Check memory profiling
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# Build and run
make build
./bin/gotsl 8443 127.0.0.1
```

---

## Files to Create/Modify

**New Files**:
- `pkg/logging/logger.go` - Structured logging
- `pkg/security/validation.go` - Path validation
- `pkg/protocol/buffer_pool.go` - Memory management

**Modified Files**:
- `pkg/server/listener.go` - Graceful shutdown, race fixes
- `pkg/client/reverse.go` - Use buffer pool
- `pkg/client/command_handlers.go` - Use path validation
- `cmd/gotsl/main.go` - Signal handling
- `go.mod` - No new dependencies for Phase 1

**Total New Code**: ~400 lines
**Estimated Time**: 3-4 hours for all 5 quick wins

---

## Verification Checklist

- [ ] All tests pass with `go test ./...`
- [ ] No race conditions with `go test -race ./...`
- [ ] Code builds with `make build`
- [ ] Shutdown works cleanly (graceful exit)
- [ ] Logs are JSON formatted
- [ ] Path validation works (test with `../` and absolute paths)
- [ ] No memory leaks with large files
- [ ] Buffer pool reduces allocations

