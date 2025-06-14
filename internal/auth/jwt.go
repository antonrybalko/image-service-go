package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// ContextKey is a custom type for context keys
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "userID"
	// TokenKey is the context key for the JWT token
	TokenKey ContextKey = "token"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	PublicKeyURL string // URL to JWKS endpoint for RS256
	Secret       string // Secret key for HS256
	Algorithm    string // "RS256" or "HS256"
}

// JWTClaims represents the expected claims in the JWT token
type JWTClaims struct {
	jwt.RegisteredClaims
	// Add any custom claims here if needed
}

// JWKS cache to avoid fetching keys on every request
var (
	jwksCache     jwk.Set
	jwksCacheMu   sync.RWMutex
	jwksCacheTime time.Time
	jwksCacheTTL  = 24 * time.Hour // Cache keys for 24 hours
)

// JWTMiddleware creates a middleware that validates JWT tokens
func JWTMiddleware(config JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			tokenString := extractTokenFromHeader(r)
			if tokenString == "" {
				http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
				return
			}

			// Parse and validate token
			claims, err := validateToken(tokenString, config)
			if err != nil {
				http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
				return
			}

			// Extract user ID from subject claim
			userID, err := claims.GetSubject()
			if err != nil || userID == "" {
				http.Error(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, TokenKey, tokenString)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check if the header starts with "Bearer "
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// validateToken validates the JWT token based on the configured algorithm
func validateToken(tokenString string, config JWTConfig) (*JWTClaims, error) {
	var claims JWTClaims

	switch config.Algorithm {
	case "RS256":
		return validateRS256Token(tokenString, config.PublicKeyURL, &claims)
	case "HS256":
		return validateHS256Token(tokenString, config.Secret, &claims)
	default:
		return nil, fmt.Errorf("unsupported JWT algorithm: %s", config.Algorithm)
	}
}

// validateRS256Token validates a token signed with RS256
func validateRS256Token(tokenString, publicKeyURL string, claims *JWTClaims) (*JWTClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kidInterface, ok := token.Header["kid"]
		if !ok {
			return nil, errors.New("no key ID in token header")
		}

		kid, ok := kidInterface.(string)
		if !ok {
			return nil, errors.New("invalid key ID format")
		}

		// Get the public key from JWKS
		publicKey, err := getPublicKeyFromJWKS(publicKeyURL, kid)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// validateHS256Token validates a token signed with HS256
func validateHS256Token(tokenString, secret string, claims *JWTClaims) (*JWTClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// getPublicKeyFromJWKS fetches and caches public keys from a JWKS endpoint
func getPublicKeyFromJWKS(jwksURL, kid string) (*rsa.PublicKey, error) {
	// Check if we need to refresh the cache
	jwksCacheMu.RLock()
	needRefresh := jwksCache == nil || time.Since(jwksCacheTime) > jwksCacheTTL
	jwksCacheMu.RUnlock()

	// Refresh the cache if needed
	if needRefresh {
		err := refreshJWKSCache(jwksURL)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh JWKS cache: %w", err)
		}
	}

	// Get the key from the cache
	jwksCacheMu.RLock()
	defer jwksCacheMu.RUnlock()

	if jwksCache == nil {
		return nil, errors.New("JWKS cache is empty")
	}

	key, found := jwksCache.LookupKeyID(kid)
	if !found {
		return nil, fmt.Errorf("key ID %s not found in JWKS", kid)
	}

	var rawKey interface{}
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to get raw key: %w", err)
	}

	publicKey, ok := rawKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("key is not an RSA public key")
	}

	return publicKey, nil
}

// refreshJWKSCache fetches the latest keys from the JWKS endpoint
func refreshJWKSCache(jwksURL string) error {
	jwksCacheMu.Lock()
	defer jwksCacheMu.Unlock()

	// Fetch the JWKS
	resp, err := http.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log the error in a real application
			_ = err
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status code %d", resp.StatusCode)
	}

	// Read the response body into a byte slice
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response body: %w", err)
	}

	// Parse the JWKS from the byte slice
	set, err := jwk.Parse(bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Update the cache
	jwksCache = set
	jwksCacheTime = time.Now()

	return nil
}

// GetUserIDFromContext extracts the user ID from the context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetTokenFromContext extracts the JWT token from the context
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(TokenKey).(string)
	return token, ok
}

// MockJWTMiddleware creates a middleware that skips JWT validation for testing
func MockJWTMiddleware(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add mock user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, TokenKey, "mock-token")

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WriteUnauthorizedResponse writes a standardized unauthorized response
func WriteUnauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		_ = err // Acknowledge the error to satisfy linter
	}
}
