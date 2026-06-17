package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	userIDKey           = "authenticated_user_id"
	invalidTokenMessage = "The access token is invalid or expired"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func IssueToken(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			utils.ResponseFailed(c, http.StatusUnauthorized, "UNAUTHORIZED", "A bearer token is required")
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		userID, err := ValidateToken(raw, secret)
		if err != nil {
			utils.ResponseFailed(c, http.StatusUnauthorized, "INVALID_TOKEN", invalidTokenMessage)
			return
		}
		c.Set(userIDKey, userID)
		c.Next()
	}
}

func ValidateToken(raw, secret string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, errors.New("invalid user claim")
	}
	return userID, nil
}

func UserID(c *gin.Context) uuid.UUID {
	value, _ := c.Get(userIDKey)
	userID, _ := value.(uuid.UUID)
	return userID
}
