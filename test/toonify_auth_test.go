package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
	"github.com/PratSins/FrameVerse-Backend/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

func TestToonifyRoutes_RequireAuth(t *testing.T) {
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

	cfg := &config.Config{
		AuthPublicKeyPEM: string(pubPEM),
		Environment:      "production", // Enforce strict validation
	}

	client, err := authclient.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create authclient: %v", err)
	}

	// 2. Set up a protected router simulating Toonify
	r := chi.NewRouter()
	authGuard := middleware.RequireAuth(client)

	reachedHandler := false
	r.Route("/api/v1/toonify", func(r chi.Router) {
		r.Use(authGuard)
		r.Post("/upload-url", func(w http.ResponseWriter, r *http.Request) {
			reachedHandler = true
			w.WriteHeader(http.StatusOK)
		})
		r.Post("/{jobID}/process", func(w http.ResponseWriter, r *http.Request) {
			reachedHandler = true
			w.WriteHeader(http.StatusAccepted)
		})
	})

	// 3. Test Case A: Unauthenticated request (no header) to upload-url
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/toonify/upload-url", nil)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)

	if wA.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated upload-url, got %d", wA.Code)
	}
	if reachedHandler {
		t.Fatalf("handler should NOT have been reached for unauthenticated upload-url")
	}

	// 4. Test Case B: Unauthenticated request to process endpoint (invoking Gemini)
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/toonify/job-abc-123/process", nil)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)

	if wB.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated process, got %d", wB.Code)
	}
	if reachedHandler {
		t.Fatalf("handler should NOT have been reached for unauthenticated process")
	}

	// 5. Test Case C: Expired token
	expiredClaims := authclient.UserClaims{
		UserID: "user-456",
		Email:  "expired@frameverse.com",
		Tier:   "free",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-10 * time.Minute)), // already expired
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS256, expiredClaims)
	expiredStr, _ := expiredToken.SignedString(privateKey)

	reqC := httptest.NewRequest(http.MethodPost, "/api/v1/toonify/job-abc-123/process", nil)
	reqC.Header.Set("Authorization", "Bearer "+expiredStr)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)

	if wC.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", wC.Code)
	}
	if reachedHandler {
		t.Fatalf("handler should NOT have been reached with expired token")
	}

	// 6. Test Case D: Valid signed token -> passes through
	validClaims := authclient.UserClaims{
		UserID: "user-123",
		Email:  "alice@frameverse.com",
		Tier:   "pro",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims)
	validStr, _ := validToken.SignedString(privateKey)

	reqD := httptest.NewRequest(http.MethodPost, "/api/v1/toonify/upload-url", nil)
	reqD.Header.Set("Authorization", "Bearer "+validStr)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)

	if wD.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid token, got %d", wD.Code)
	}
	if !reachedHandler {
		t.Fatalf("handler should have been reached with valid token")
	}
}
