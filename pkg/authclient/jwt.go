package authclient

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	jwt.RegisteredClaims
}

type JWKSResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

type Client struct {
	jwksURL     string
	publicKey   *rsa.PublicKey
	httpClient  *http.Client
	lastFetched time.Time
	cacheTTL    time.Duration
	environment string
	mu          sync.RWMutex
}

func NewClient(cfg *config.Config) (*Client, error) {
	c := &Client{
		jwksURL:     cfg.AuthJWKSURL,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		cacheTTL:    15 * time.Minute,
		environment: cfg.Environment,
	}

	if cfg.AuthPublicKeyPEM != "" {
		pubKey, err := parseRSAPublicKeyFromPEM([]byte(cfg.AuthPublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse static public key: %w", err)
		}
		c.publicKey = pubKey
	}

	return c, nil
}

func (c *Client) ValidateToken(tokenString string) (*UserClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("empty token")
	}

	// Try verifying with cached or fetched public key
	pubKey, err := c.getPublicKey()
	if err != nil {
		// In development mode, if auth service is offline and no key is configured, allow mock/dev token parsing
		if c.environment == "development" {
			claims, devErr := parseUnverifiedToken(tokenString)
			if devErr == nil {
				return claims, nil
			}
		}
		return nil, fmt.Errorf("failed to retrieve auth public key: %w", err)
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func (c *Client) getPublicKey() (*rsa.PublicKey, error) {
	c.mu.RLock()
	if c.publicKey != nil && time.Since(c.lastFetched) < c.cacheTTL {
		defer c.mu.RUnlock()
		return c.publicKey, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.publicKey != nil && time.Since(c.lastFetched) < c.cacheTTL {
		return c.publicKey, nil
	}

	if c.jwksURL == "" {
		if c.publicKey != nil {
			return c.publicKey, nil
		}
		return nil, errors.New("no JWKS URL or static public key configured")
	}

	resp, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		if c.publicKey != nil {
			return c.publicKey, nil // fallback to stale key
		}
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if c.publicKey != nil {
			return c.publicKey, nil
		}
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, errors.New("no keys found in JWKS response")
	}

	key := jwks.Keys[0]
	pubKey, err := parseRSAPublicKeyFromJWK(key.N, key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to construct public key from JWK: %w", err)
	}

	c.publicKey = pubKey
	c.lastFetched = time.Now()
	return c.publicKey, nil
}

func parseRSAPublicKeyFromJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	var eInt int
	for _, b := range eBytes {
		eInt = (eInt << 8) | int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: eInt,
	}, nil
}

func parseRSAPublicKeyFromPEM(keyData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pubKey, nil
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return pubKey, nil
}

func parseUnverifiedToken(tokenString string) (*UserClaims, error) {
	parser := jwt.NewParser()
	claims := &UserClaims{}
	_, _, err := parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
