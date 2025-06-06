package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys
const (
	UserIDKey contextKey = "userID"
)

// Config holds JWT authentication configuration
type Config struct {
	PublicKeyURL string
	Secret       string
	Algorithm    string
}

// JWTMiddleware handles JWT authentication
type JWTMiddleware struct {
	config      Config
	logger      *zap.SugaredLogger
	publicKey   interface{}
	keyLock     sync.RWMutex
	keyFetchedAt time.Time
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(config Config, logger *zap.SugaredLogger) *JWTMiddleware {
	return &JWTMiddleware{
		config: config,
		logger: logger,
	}
}

// Middleware returns a chi middleware function for JWT authentication
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		tokenString := extractTokenFromHeader(r)
		if tokenString == "" {
			m.unauthorized(w, r, errors.New("no authorization token provided"))
			return
		}

		// Parse and validate token
		userID, err := m.validateToken(tokenString)
		if err != nil {
			m.unauthorized(w, r, err)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(r *http.Request) string {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check if it's a Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// validateToken validates the JWT token and returns the user ID
func (m *JWTMiddleware) validateToken(tokenString string) (string, error) {
	switch strings.ToUpper(m.config.Algorithm) {
	case "RS256":
		return m.validateRS256Token(tokenString)
	case "HS256":
		return m.validateHS256Token(tokenString)
	default:
		return "", fmt.Errorf("unsupported JWT algorithm: %s", m.config.Algorithm)
	}
}

// validateHS256Token validates a token signed with HS256 algorithm
func (m *JWTMiddleware) validateHS256Token(tokenString string) (string, error) {
	if m.config.Secret == "" {
		return "", errors.New("JWT secret not configured")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return extractUserIDFromToken(token)
}

// validateRS256Token validates a token signed with RS256 algorithm
func (m *JWTMiddleware) validateRS256Token(tokenString string) (string, error) {
	// Get the public key
	publicKey, err := m.getPublicKey()
	if err != nil {
		return "", err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return extractUserIDFromToken(token)
}

// extractUserIDFromToken extracts the user ID from the token claims
func extractUserIDFromToken(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	// Extract the subject (user ID)
	sub, ok := claims["sub"]
	if !ok {
		return "", errors.New("token missing 'sub' claim")
	}

	userID, ok := sub.(string)
	if !ok {
		return "", errors.New("'sub' claim is not a string")
	}

	return userID, nil
}

// getPublicKey fetches and caches the public key from the JWKS endpoint
func (m *JWTMiddleware) getPublicKey() (interface{}, error) {
	// Check if we already have a cached key
	m.keyLock.RLock()
	if m.publicKey != nil && time.Since(m.keyFetchedAt) < 1*time.Hour {
		defer m.keyLock.RUnlock()
		return m.publicKey, nil
	}
	m.keyLock.RUnlock()

	// Need to fetch or refresh the key
	m.keyLock.Lock()
	defer m.keyLock.Unlock()

	// Double-check after acquiring the write lock
	if m.publicKey != nil && time.Since(m.keyFetchedAt) < 1*time.Hour {
		return m.publicKey, nil
	}

	if m.config.PublicKeyURL == "" {
		return nil, errors.New("JWT public key URL not configured")
	}

	// Fetch the JWKS
	resp, err := http.Get(m.config.PublicKeyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: HTTP %d", resp.StatusCode)
	}

	// Parse the JWKS
	var jwks struct {
		Keys []struct {
			Kid string   `json:"kid"`
			Kty string   `json:"kty"`
			Use string   `json:"use"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			X5c []string `json:"x5c"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Find the first RSA key for signature verification
	for _, key := range jwks.Keys {
		if key.Kty == "RSA" && (key.Use == "sig" || key.Use == "") {
			// Parse the key
			publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----", key.X5c[0])))
			if err != nil {
				m.logger.Warnw("Failed to parse RSA public key", "error", err)
				continue
			}

			m.publicKey = publicKey
			m.keyFetchedAt = time.Now()
			return publicKey, nil
		}
	}

	return nil, errors.New("no suitable key found in JWKS")
}

// unauthorized responds with a 401 Unauthorized status
func (m *JWTMiddleware) unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	m.logger.Debugw("Unauthorized request", "error", err, "path", r.URL.Path)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Unauthorized",
	})
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// RequireAuth is a helper middleware that ensures a user ID is present
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok || userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Authentication required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
