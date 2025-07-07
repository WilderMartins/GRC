package auth

import (
	"fmt"
	"net/http"
	"os"
	"phoenixgrc/backend/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtKey []byte

// Claims struct to be encoded to JWT
type Claims struct {
	UserID         uuid.UUID      `json:"user_id"`
	OrganizationID uuid.UUID      `json:"org_id"`
	Email          string         `json:"email"`
	Role           models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// InitializeJWT loads the JWT secret key from environment variables.
func InitializeJWT() error {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}
	jwtKey = []byte(secret)
	return nil
}

// GenerateToken generates a new JWT token for a given user.
func GenerateToken(user *models.User, organizationID uuid.UUID) (string, error) {
	if len(jwtKey) == 0 {
		return "", fmt.Errorf("JWT secret key not initialized. Call InitializeJWT() first")
	}

	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours
	// Potentially make lifespan configurable via env var
	tokenLifespanStr := os.Getenv("JWT_TOKEN_LIFESPAN_HOURS")
	if tokenLifespanHours, err := time.ParseDuration(tokenLifespanStr + "h"); err == nil {
		expirationTime = time.Now().Add(tokenLifespanHours)
	}


	claims := &Claims{
		UserID:         user.ID,
		OrganizationID: organizationID,
		Email:          user.Email,
		Role:           user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "phoenix-grc", // Optional: identify the issuer
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token string.
// Returns the claims if the token is valid, otherwise returns an error.
func ValidateToken(tokenString string) (*Claims, error) {
	if len(jwtKey) == 0 {
		return nil, fmt.Errorf("JWT secret key not initialized")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AuthMiddleware creates a Gin middleware for JWT authentication.
// It checks for a valid JWT in the Authorization header (Bearer token).
// If valid, it sets the user's claims in the Gin context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		tokenString := parts[1]

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		// Store claims in context for use by handlers
		c.Set("userID", claims.UserID)
		c.Set("organizationID", claims.OrganizationID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Set("claims", claims) // Or set the whole claims struct

		c.Next()
	}
}
