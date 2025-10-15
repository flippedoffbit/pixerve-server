package utils

import (
	"errors"
	"fmt"
	"time"

	"pixerve/models"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

var (
	ErrInvalidToken     = errors.New("invalid token format")
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenNotYetValid = errors.New("token not yet valid")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrInvalidIssuer    = errors.New("invalid issuer")
)

// VerifyConfig holds verification configuration
type VerifyConfig struct {
	SecretKey      []byte        // For HMAC (HS256)
	PublicKey      any           // For RSA (RS256) - *rsa.PublicKey
	ExpectedIssuer string        // Optional: validate issuer
	ClockSkew      time.Duration // Optional: allow clock skew (default 0)
}

// VerifyPixerveJWT safely verifies and decodes a Pixerve JWT
func VerifyPixerveJWT(tokenString string, config VerifyConfig) (*models.PixerveJWT, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Determine which algorithms to accept based on config
	var allowedAlgs []jose.SignatureAlgorithm
	if config.SecretKey != nil {
		allowedAlgs = append(allowedAlgs, jose.HS256)
	}
	if config.PublicKey != nil {
		allowedAlgs = append(allowedAlgs, jose.RS256)
	}

	if len(allowedAlgs) == 0 {
		return nil, errors.New("no verification key provided")
	}

	// Parse the token
	tok, err := jwt.ParseSigned(tokenString, allowedAlgs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Prepare claims struct
	claims := &models.PixerveJWT{}

	// Verify signature and extract claims
	var verifyErr error
	if config.SecretKey != nil {
		verifyErr = tok.Claims(config.SecretKey, claims)
	} else if config.PublicKey != nil {
		verifyErr = tok.Claims(config.PublicKey, claims)
	}

	if verifyErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignature, verifyErr)
	}

	// Validate timestamps
	now := time.Now().Unix()
	clockSkew := int64(config.ClockSkew.Seconds())

	// Check expiry
	if claims.ExpiresAt > 0 && claims.ExpiresAt < (now-clockSkew) {
		return nil, ErrTokenExpired
	}

	// Check issued at (not in future)
	if claims.IssuedAt > 0 && claims.IssuedAt > (now+clockSkew) {
		return nil, ErrTokenNotYetValid
	}

	// Validate issuer if specified
	if config.ExpectedIssuer != "" && claims.Issuer != config.ExpectedIssuer {
		return nil, fmt.Errorf("%w: expected '%s', got '%s'",
			ErrInvalidIssuer, config.ExpectedIssuer, claims.Issuer)
	}

	return claims, nil
}

// CreatePixerveJWT creates a signed JWT from PixerveJWT claims
func CreatePixerveJWT(claims *models.PixerveJWT) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}

	// Use HMAC signing with a default secret key for testing
	// In production, this should be configurable
	secretKey := []byte("test-secret-key-for-jwt-signing-at-least-32-bytes-long")

	// Create the signer
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: secretKey}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	// Create the JWT
	builder := jwt.Signed(signer)
	builder = builder.Claims(claims)
	token, err := builder.Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to create JWT: %w", err)
	}

	return token, nil
}

// Example usage:
/*
func ExampleUsage() {
	// HMAC verification
	claims, err := VerifyPixerveJWT(token, VerifyConfig{
		SecretKey:      []byte("your-secret-key"),
		ExpectedIssuer: "pixerve-api",
		ClockSkew:      time.Minute * 5,
	})

	// RSA verification
	claims, err := VerifyPixerveJWT(token, VerifyConfig{
		PublicKey:      publicKey, // *rsa.PublicKey
		ExpectedIssuer: "pixerve-api",
	})
}
*/
