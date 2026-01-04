package middleware

import (
	"context"
	"crypto/rsa"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
)

type cacheEntry struct {
	claims jwt.MapClaims
	exp    int64
}

type AuthMiddleware struct {
	publicKey   *rsa.PublicKey
	cache       sync.Map
	redisClient *redis.Client
	redisCB     *gobreaker.CircuitBreaker
}

const CacheCleanupInterval = 10 * time.Minute

func NewAuthMiddleware(publicKey *rsa.PublicKey, redisClient *redis.Client) *AuthMiddleware {
	// Configure circuit breaker for Redis operations
	// Fail-closed strategy: When circuit is open, reject requests for security
	redisCB := config.NewCircuitBreaker("Redis-Auth")

	m := &AuthMiddleware{
		publicKey:   publicKey,
		redisClient: redisClient,
		redisCB:     redisCB,
	}

	// Start Background Janitor to sweep L1 cache every 10 minutes
	go m.startJanitor(CacheCleanupInterval)

	return m
}

type contextKey string

const (
	UserIDKey contextKey = "userID"
	RoleKey   contextKey = "role"
	TokenKey  contextKey = "token"
)

func (m *AuthMiddleware) RequireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // start time for processing time measurement

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("Missing Authorization header")
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Peek and L1 cache check
		claims, jti, err := m.getClaimsFromCacheOrParse(tokenString)
		if err != nil {
			log.Printf("Token parse error: %v", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// L2 Redis blacklist check (Kill-Switch)
		revoked, err := m.isBlacklisted(jti, r.Context())
		if err != nil {
			// Circuit breaker is open or Redis failed - FAIL CLOSED
			log.Printf("[CRITICAL] Authentication service unavailable: %v", err)
			http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
			return
		}
		if revoked {
			m.cache.Delete(jti)
			log.Printf("Rejected: JTI %s is blacklisted", jti)
			http.Error(w, "token revoked", http.StatusUnauthorized)
			return
		}

		userRole, _ := claims["role"].(string)
		if !m.isAuthorized(userRole, roles) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		userID, _ := claims["sub"].(string)

		log.Printf("Token validated - UserID: %s, Role: %s", userID, userRole)

		allowedRoles := false
		for _, r := range roles {
			if userRole == r {
				allowedRoles = true
				break
			}
		}
		if !allowedRoles {
			log.Printf("Role mismatch: required one of %v, got %s", roles, userRole)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, RoleKey, userRole)
		ctx = context.WithValue(ctx, TokenKey, tokenString)

		log.Printf("AuthMiddleware processing time: %v", time.Since(start))

		next(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) getClaimsFromCacheOrParse(tokenString string) (jwt.MapClaims, string, error) {
	// Peek at the JTI without verifying the signature yet
	parser := new(jwt.Parser)
	unverifiedToken, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, "", err
	}

	claims, _ := unverifiedToken.Claims.(jwt.MapClaims)
	jti, _ := claims["jti"].(string)
	expFloat, _ := claims["exp"].(float64)
	exp := int64(expFloat)

	if jti == "" {
		return nil, "", errors.New("missing jti")
	}

	// Immediate expiry check (Fastest fail)
	if time.Now().Unix() > exp {
		return nil, "", errors.New("token expired")
	}

	// L1 Cache Lookup (Keyed by JTI)
	if entry, ok := m.cache.Load(jti); ok {
		return entry.(cacheEntry).claims, jti, nil
	}

	// Full RSA Validation (Cold path)
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, "", err
	}

	// Store in Cache
	m.cache.Store(jti, cacheEntry{claims: claims, exp: exp})

	return claims, jti, nil
}

func (m *AuthMiddleware) isBlacklisted(jti string, ctx context.Context) (bool, error) {
	// Execute Redis check through circuit breaker
	result, err := m.redisCB.Execute(func() (interface{}, error) {
		return m.redisClient.Exists(ctx, "blacklist:"+jti).Result()
	})
	
	if err != nil {
		// Circuit breaker is open or Redis operation failed
		log.Printf("[CRITICAL] Redis blacklist check failed (circuit may be open): %v", err)
		return false, err
	}
	
	isRevoked, ok := result.(int64)
	return ok && isRevoked > 0, nil
}

func (m *AuthMiddleware) isAuthorized(userRole string, allowedRoles []string) bool {
	for _, r := range allowedRoles {
		if userRole == r {
			return true
		}
	}
	return false
}

func (m *AuthMiddleware) startJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		deleted := 0
		m.cache.Range(func(key, value any) bool {
			if entry, ok := value.(cacheEntry); ok && now > entry.exp {
				m.cache.Delete(key)
				deleted++
			}
			return true
		})
		if deleted > 0 {
			log.Printf("L1 Janitor: Purged %d expired entries", deleted)
		}
	}
}
