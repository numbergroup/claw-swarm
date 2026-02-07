package types

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	jwt.RegisteredClaims
	IsBot      bool   `json:"isBot"`
	UserID     string `json:"userId,omitempty"`
	BotSpaceID string `json:"botSpaceId,omitempty"`
	BotID      string `json:"botId,omitempty"`
	IsManager  bool   `json:"isManager,omitempty"`
}
