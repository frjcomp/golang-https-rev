package certs

import (
	"encoding/hex"
	"testing"
	"time"
)

// TestGenerateSelfSignedCert tests certificate generation
func TestGenerateSelfSignedCert(t *testing.T) {
	cert, fingerprint, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("Certificate has no data")
	}

	// Fingerprint should be 64 hex characters (SHA256)
	if len(fingerprint) != 64 {
		t.Errorf("fingerprint should be 64 characters, got %d", len(fingerprint))
	}

	t.Log("✓ Self-signed certificate generated successfully")
}

// TestCertificateValidity tests certificate has required properties
func TestCertificateValidity(t *testing.T) {
	cert, _, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("Certificate data is empty")
	}

	if cert.PrivateKey == nil {
		t.Fatal("Private key is nil")
	}

	t.Log("✓ Certificate is valid and contains required data")
}

// TestMultipleCertificates tests that multiple certificates can be generated
func TestMultipleCertificates(t *testing.T) {
	cert1, _, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate first certificate: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	cert2, _, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate second certificate: %v", err)
	}

	// Certificates should be different (different serial numbers)
	if string(cert1.Certificate[0]) == string(cert2.Certificate[0]) {
		t.Fatal("Two generated certificates are identical")
	}

	t.Log("✓ Multiple unique certificates can be generated")
}

// TestGenerateSecret tests shared secret generation
func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}

	// Should be 64 hex characters (32 bytes encoded as hex)
	if len(secret) != 64 {
		t.Errorf("Expected 64 character hex string, got %d characters", len(secret))
	}

	// Should be valid hex
	_, err = hex.DecodeString(secret)
	if err != nil {
		t.Errorf("Secret is not valid hex: %v", err)
	}

	t.Log("✓ Secret generation successful")
}

// TestGenerateSecretUniqueness tests that secrets are unique
func TestGenerateSecretUniqueness(t *testing.T) {
	secret1, err := GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate first secret: %v", err)
	}

	secret2, err := GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate second secret: %v", err)
	}

	if secret1 == secret2 {
		t.Error("Two generated secrets are identical, should be random")
	}

	t.Log("✓ Secrets are unique")
}

// TestGenerateSecretRandomness tests basic randomness properties
func TestGenerateSecretRandomness(t *testing.T) {
	// Generate multiple secrets and check they're all different
	secrets := make(map[string]bool)
	for i := 0; i < 10; i++ {
		secret, err := GenerateSecret()
		if err != nil {
			t.Fatalf("Failed to generate secret %d: %v", i, err)
		}

		if secrets[secret] {
			t.Errorf("Duplicate secret generated: %s", secret)
		}
		secrets[secret] = true
	}

	t.Log("✓ Secrets show good randomness")
}

// TestGetCertificateFingerprint tests certificate fingerprint calculation
func TestGetCertificateFingerprint(t *testing.T) {
	cert, _, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	fingerprint, err := GetCertificateFingerprint(cert)
	if err != nil {
		t.Fatalf("Failed to get fingerprint: %v", err)
	}

	// Should be 64 hex characters (SHA256)
	if len(fingerprint) != 64 {
		t.Errorf("Expected 64 character fingerprint, got %d", len(fingerprint))
	}

	// Should be valid hex
	_, err = hex.DecodeString(fingerprint)
	if err != nil {
		t.Errorf("Fingerprint is not valid hex: %v", err)
	}

	t.Log("✓ Certificate fingerprint calculated successfully")
}

// TestGetCertificateFingerprintConsistency tests that fingerprint is consistent
func TestGetCertificateFingerprintConsistency(t *testing.T) {
	cert, _, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	fingerprint1, err := GetCertificateFingerprint(cert)
	if err != nil {
		t.Fatalf("Failed to get first fingerprint: %v", err)
	}

	fingerprint2, err := GetCertificateFingerprint(cert)
	if err != nil {
		t.Fatalf("Failed to get second fingerprint: %v", err)
	}

	if fingerprint1 != fingerprint2 {
		t.Error("Fingerprints should be identical for the same certificate")
	}

	t.Log("✓ Certificate fingerprint is consistent")
}

// TestGetCertificateFingerprintMatchesGeneratedFingerprint tests consistency between methods
func TestGetCertificateFingerprintMatchesGeneratedFingerprint(t *testing.T) {
	cert, generatedFingerprint, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	calculatedFingerprint, err := GetCertificateFingerprint(cert)
	if err != nil {
		t.Fatalf("Failed to calculate fingerprint: %v", err)
	}

	if generatedFingerprint != calculatedFingerprint {
		t.Errorf("Fingerprints don't match:\nGenerated: %s\nCalculated: %s",
			generatedFingerprint, calculatedFingerprint)
	}

	t.Log("✓ Fingerprints match between generation methods")
}

// TestDifferentCertsDifferentFingerprints tests that different certs have different fingerprints
func TestDifferentCertsDifferentFingerprints(t *testing.T) {
	cert1, fingerprint1, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate first certificate: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different serial number

	cert2, fingerprint2, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate second certificate: %v", err)
	}

	if fingerprint1 == fingerprint2 {
		t.Error("Different certificates should have different fingerprints")
	}

	// Double check with GetCertificateFingerprint
	fp1, _ := GetCertificateFingerprint(cert1)
	fp2, _ := GetCertificateFingerprint(cert2)

	if fp1 == fp2 {
		t.Error("Different certificates should have different fingerprints (via GetCertificateFingerprint)")
	}

	t.Log("✓ Different certificates have different fingerprints")
}
