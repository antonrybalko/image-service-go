package auth

import (
	"context"
	"net/http"
	"sync"
)

// MockJWTMiddleware is a test implementation of JWT middleware
// that allows injecting user IDs without real token validation
type MockJWTMiddleware struct {
	mu            sync.RWMutex
	defaultUserID string
	userIDMap     map[string]string // Maps token to user ID
	calls         int
}

// NewMockJWTMiddleware creates a new mock JWT middleware for testing
func NewMockJWTMiddleware(defaultUserID string) *MockJWTMiddleware {
	return &MockJWTMiddleware{
		defaultUserID: defaultUserID,
		userIDMap:     make(map[string]string),
	}
}

// Middleware returns a chi middleware function that adds the configured user ID to the context
func (m *MockJWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.calls++
		m.mu.Unlock()

		// Extract token from header (to support token-specific user IDs)
		token := extractTokenFromHeader(r)
		
		// Determine which user ID to use
		var userID string
		
		m.mu.RLock()
		if token != "" && m.userIDMap[token] != "" {
			// Use token-specific mapping if available
			userID = m.userIDMap[token]
		} else {
			// Fall back to default user ID
			userID = m.defaultUserID
		}
		m.mu.RUnlock()

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetDefaultUserID changes the default user ID returned by the middleware
func (m *MockJWTMiddleware) SetDefaultUserID(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultUserID = userID
}

// SetUserIDForToken maps a specific token to a user ID
func (m *MockJWTMiddleware) SetUserIDForToken(token, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userIDMap[token] = userID
}

// GetCallCount returns the number of times the middleware was called
func (m *MockJWTMiddleware) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls
}

// Reset resets the middleware state
func (m *MockJWTMiddleware) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = 0
	m.userIDMap = make(map[string]string)
}
