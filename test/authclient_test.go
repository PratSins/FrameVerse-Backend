package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthClient_ValidateToken(t *testing.T) {
	// 1. Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// 2. Initialize AuthClient with static public key
	cfg := &config.Config{
		AuthPublicKeyPEM: string(pubPEM),
		Environment:      "development",
	}

	client, err := authclient.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create authclient: %v", err)
	}

	// 3. Generate a signed token using the private key
	claims := authclient.UserClaims{
		UserID: "test-user-123",
		Email:  "test@frameverse.com",
		Tier:   "pro",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "frameverse-auth",
			Subject:   "test-user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// 4. Validate the token using authclient
	validatedClaims, err := client.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if validatedClaims.UserID != "test-user-123" {
		t.Errorf("expected UserID test-user-123, got %s", validatedClaims.UserID)
	}

	if validatedClaims.Email != "test@frameverse.com" {
		t.Errorf("expected Email test@frameverse.com, got %s", validatedClaims.Email)
	}

	if validatedClaims.Tier != "pro" {
		t.Errorf("expected Tier pro, got %s", validatedClaims.Tier)
	}
}
