package routes

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/numbergroup/claw-swarm/pkg/types"
)

type authMiddleware struct {
	jwtSecret []byte
}

func (am *authMiddleware) Handle(c *gin.Context) {
	tokenStr := ""

	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		tokenStr = strings.TrimPrefix(auth, "Bearer ")
	}

	if tokenStr == "" {
		tokenStr = c.Query("token")
	}

	if tokenStr == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
		return
	}

	claims := &types.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return am.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	c.Set("claims", claims)
	c.Next()
}
