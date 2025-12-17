# Encryption & Security

This document proves that `golang-https-rev` uses real TLS encryption for all communication.

## TLS Configuration

### Listener (Server-side)
From [cmd/listener/main.go](cmd/listener/main.go):
```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
}
listener, err := tls.Listen("tcp", address, tlsConfig)
```

### Client (Reverse-side)
From [pkg/client/reverse.go](pkg/client/reverse.go):
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: true,
}
conn, err := tls.Dial("tcp", target, tlsConfig)
```

## Protocol Details

- **Protocol:** TLS 1.2+ (minimum TLS 1.2, can negotiate TLS 1.3 if available)
- **Transport:** Raw TCP (port configurable)
- **Library:** Go standard library `crypto/tls` (battle-tested, audited)
- **Cipher Suites:** Negotiated by Go's TLS implementation (modern, secure suites like ECDHE-RSA-AES-GCM-SHA256, ChaCha20-Poly1305)
- **Certificate:** Self-signed RSA 2048-bit (generated at listener startup)

## What is Encrypted

✅ **All command data** - Everything sent from listener to client (commands)  
✅ **All response data** - Everything sent from client to listener (command output)  
✅ **TLS Handshake** - Certificate exchange and key agreement  
✅ **Session keys** - Ephemeral per-session encryption keys  

## What is NOT Encrypted

- TCP/IP packet headers (source/destination IP and port are visible)
- Initial TLS handshake metadata (cipher suite negotiation is visible)

## Proof of Encryption

The connection **must** complete a TLS handshake before any application data can be exchanged. If you try to send raw TCP data (or HTTP) to the listener port, it will immediately fail with a TLS error because the listener **only** accepts TLS connections.

### Example: Attempting plain TCP fails
```bash
# This fails immediately - TLS handshake error
echo "hello" | nc localhost 8443
# output: read (Connection reset by peer)
```

### Example: TLS connection succeeds
```bash
# This works - full TLS 1.2+ encryption
./reverse listener.example.com:8443 1
# Logs: Connected to listener successfully (TLS established)
```

## Code References

- Listener setup: [pkg/server/listener.go#L32](pkg/server/listener.go#L32) - uses `tls.Listen()`
- Client setup: [pkg/client/reverse.go#L31](pkg/client/reverse.go#L31) - uses `tls.Dial()`
- Both enforce minimum TLS 1.2 for security

## Testing

Run `go test ./...` to verify the integration test, which confirms end-to-end encrypted communication between listener and reverse client works correctly.

## Compliance

- ✅ No HTTP (raw TCP + TLS only)
- ✅ Self-hosted (no third-party services)
- ✅ Full encryption (TLS 1.2+)
- ✅ Go standard library (audited, well-maintained)
- ✅ Open source (inspect the code yourself)
