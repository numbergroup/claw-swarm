package routes

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/numbergroup/claw-swarm/pkg/config"
	"github.com/numbergroup/claw-swarm/pkg/db"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/claw-swarm/pkg/ws"
	"github.com/numbergroup/server"
	"github.com/sirupsen/logrus"
)

type RouteHandler struct {
	log           logrus.Ext1FieldLogger
	conf          *config.Config
	userDB        db.UserDB
	botSpaceDB    db.BotSpaceDB
	spaceMemberDB db.SpaceMemberDB
	botDB         db.BotDB
	messageDB     db.MessageDB
	botStatusDB   db.BotStatusDB
	summaryDB     db.SummaryDB
	inviteCodeDB  db.InviteCodeDB
	botSkillDB    db.BotSkillDB
	auth          *authMiddleware
	hub           *ws.Hub
}

func NewRouteHandler(
	conf *config.Config,
	userDB db.UserDB,
	botSpaceDB db.BotSpaceDB,
	spaceMemberDB db.SpaceMemberDB,
	botDB db.BotDB,
	messageDB db.MessageDB,
	botStatusDB db.BotStatusDB,
	summaryDB db.SummaryDB,
	inviteCodeDB db.InviteCodeDB,
	botSkillDB db.BotSkillDB,
	hub *ws.Hub,
) *RouteHandler {
	return &RouteHandler{
		log:           conf.GetLogger(),
		conf:          conf,
		userDB:        userDB,
		botSpaceDB:    botSpaceDB,
		spaceMemberDB: spaceMemberDB,
		botDB:         botDB,
		messageDB:     messageDB,
		botStatusDB:   botStatusDB,
		summaryDB:     summaryDB,
		inviteCodeDB:  inviteCodeDB,
		botSkillDB:    botSkillDB,
		auth:          &authMiddleware{jwtSecret: []byte(conf.JWTSecret)},
		hub:           hub,
	}
}

func (rh *RouteHandler) ApplyRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	{ // public auth
		api.POST("/auth/signup", rh.Signup)
		api.GET("/auth/signup-enabled", rh.SignupEnabled)
		api.POST("/auth/login", rh.Login)
		api.POST("/auth/bots/register", rh.RegisterBot)
	}

	auth := api.Group("", rh.auth.Handle, rh.trackBotLastSeen)

	{ // authenticated auth
		auth.PUT("/auth/password", rh.ChangePassword)
		auth.GET("/auth/me", rh.GetMe)
		auth.POST("/auth/refresh", rh.Refresh)
		auth.POST("/auth/bots/refresh", rh.RefreshBot)
	}

	{ // bot spaces
		auth.POST("/bot-spaces", rh.CreateBotSpace)
		auth.GET("/bot-spaces", rh.ListBotSpaces)
		auth.POST("/bot-spaces/join", rh.JoinBotSpace)

		space := auth.Group("/bot-spaces/:botSpaceId")
		space.GET("", rh.GetBotSpace)
		space.PUT("", rh.UpdateBotSpace)
		space.DELETE("", rh.DeleteBotSpace)
		space.POST("/join-codes/regenerate", rh.RegenerateJoinCodes)

		// invite codes
		space.POST("/invite-codes", rh.CreateInviteCode)
		space.GET("/invite-codes", rh.ListInviteCodes)
		space.DELETE("/invite-codes/:codeId", rh.RevokeInviteCode)

		// members
		space.GET("/members", rh.ListMembers)
		space.DELETE("/members/:userId", rh.RemoveMember)

		// bots
		space.GET("/bots", rh.ListBots)
		space.GET("/bots/:botId", rh.GetBot)
		space.DELETE("/bots/:botId", rh.RemoveBot)
		space.PUT("/bots/:botId/manager", rh.AssignManager)
		space.DELETE("/bots/:botId/manager", rh.RemoveManagerRole)

		// messages
		space.POST("/messages", rh.PostMessage)
		space.GET("/messages", rh.ListMessages)
		space.GET("/messages/since/:messageId", rh.GetMessagesSince)
		space.GET("/messages/ws", rh.SubscribeMessages)

		// statuses
		space.GET("/statuses", rh.ListStatuses)
		space.PUT("/statuses", rh.BulkUpdateStatuses)
		space.GET("/statuses/:botId", rh.GetBotStatus)
		space.PUT("/statuses/:botId", rh.UpdateBotStatus)

		// summary
		space.GET("/summary", rh.GetSummary)
		space.PUT("/summary", rh.UpdateSummary)

		// overall
		space.GET("/overall", rh.GetOverall)

		// skills
		space.POST("/skills", rh.CreateBotSkill)
		space.GET("/skills", rh.ListBotSkills)
		space.PUT("/skills/:skillId", rh.UpdateBotSkill)
		space.DELETE("/skills/:skillId", rh.DeleteBotSkill)
	}
}

func (rh *RouteHandler) getClaims(c *gin.Context) *types.Claims {
	val, exists := c.Get("claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return nil
	}
	claims, ok := val.(*types.Claims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authentication"})
		return nil
	}
	return claims
}

func (rh *RouteHandler) requireSpaceAccess(c *gin.Context) (*types.Claims, string, bool) {
	claims := rh.getClaims(c)
	if claims == nil {
		return nil, "", false
	}
	id, err := server.GetUUIDParam(c, "botSpaceId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botSpaceId"})
		return nil, "", false
	}
	botSpaceID := id.String()

	if claims.IsBot {
		if claims.BotSpaceID != botSpaceID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bot does not belong to this space"})
			return nil, "", false
		}
		return claims, botSpaceID, true
	}

	isMember, err := rh.spaceMemberDB.IsMember(c, botSpaceID, claims.UserID)
	if err != nil {
		rh.log.WithError(err).Error("failed to check membership")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check membership"})
		return nil, "", false
	}
	if !isMember {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not a member of this space"})
		return nil, "", false
	}

	return claims, botSpaceID, true
}

func (rh *RouteHandler) requireOwner(c *gin.Context) (*types.Claims, string, bool) {
	claims := rh.getClaims(c)
	if claims == nil {
		return nil, "", false
	}
	if claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bots cannot perform this action"})
		return nil, "", false
	}

	id, err := server.GetUUIDParam(c, "botSpaceId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botSpaceId"})
		return nil, "", false
	}
	botSpaceID := id.String()

	space, err := rh.botSpaceDB.GetByID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get bot space")
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot space not found"})
		return nil, "", false
	}

	if space.OwnerID != claims.UserID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the owner can perform this action"})
		return nil, "", false
	}

	return claims, botSpaceID, true
}

func (rh *RouteHandler) requireManagerBot(c *gin.Context) (*types.Claims, string, bool) {
	claims := rh.getClaims(c)
	if claims == nil {
		return nil, "", false
	}
	if !claims.IsBot || !claims.IsManager {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only manager bots can perform this action"})
		return nil, "", false
	}

	id, err := server.GetUUIDParam(c, "botSpaceId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botSpaceId"})
		return nil, "", false
	}
	botSpaceID := id.String()

	if claims.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bot does not belong to this space"})
		return nil, "", false
	}

	return claims, botSpaceID, true
}

func (rh *RouteHandler) generateCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (rh *RouteHandler) generateToken(claims *types.Claims, expiration *time.Duration) (string, error) {
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	if expiration != nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(*expiration))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(rh.conf.JWTSecret))
}
