package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthRequired validates the JWT and sets user_id in context. No DB lookup needed —
// the token's expiry and signature are sufficient; deleted-user tokens expire within 24h.
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		uidStr, ok := claims["user_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token payload"})
			return
		}

		if _, err := uuid.Parse(uidStr); err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid user id in token"})
			return
		}

		c.Set("user_id", uidStr)
		c.Next()
	}
}
