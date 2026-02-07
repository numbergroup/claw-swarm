package routes

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
	"golang.org/x/crypto/bcrypt"
)

func (rh *RouteHandler) Signup(c *gin.Context) {
	var req types.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		rh.log.WithError(err).Error("failed to hash password")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	now := time.Now()
	user := types.User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = rh.userDB.Insert(c, user)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "email already in use"})
			return
		}
		rh.log.WithError(err).Error("failed to insert user")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	exp := rh.conf.JWTExpiration
	token, err := rh.generateToken(&types.Claims{
		IsBot:  false,
		UserID: user.ID,
	}, &exp)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate token")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	c.JSON(http.StatusCreated, types.AuthResponse{Token: token, User: user})
}

func (rh *RouteHandler) Login(c *gin.Context) {
	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := rh.userDB.GetByEmail(c, strings.ToLower(req.Email))
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		rh.log.WithError(err).Error("failed to get user")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	exp := rh.conf.JWTExpiration
	token, err := rh.generateToken(&types.Claims{
		IsBot:  false,
		UserID: user.ID,
	}, &exp)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate token")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	c.JSON(http.StatusOK, types.AuthResponse{Token: token, User: user})
}

func (rh *RouteHandler) ChangePassword(c *gin.Context) {
	claims := rh.getClaims(c)
	if claims == nil {
		return
	}
	if claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bots cannot change passwords"})
		return
	}

	var req types.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := rh.userDB.GetByID(c, claims.UserID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get user")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid old password"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		rh.log.WithError(err).Error("failed to hash password")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	if err := rh.userDB.UpdatePassword(c, claims.UserID, string(hash)); err != nil {
		rh.log.WithError(err).Error("failed to update password")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (rh *RouteHandler) GetMe(c *gin.Context) {
	claims := rh.getClaims(c)
	if claims == nil {
		return
	}

	if claims.IsBot {
		bot, err := rh.botDB.GetByID(c, claims.BotID)
		if err != nil {
			rh.log.WithError(err).Error("failed to get bot")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get identity"})
			return
		}
		c.JSON(http.StatusOK, bot)
		return
	}

	user, err := rh.userDB.GetByID(c, claims.UserID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get user")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get identity"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (rh *RouteHandler) RegisterBot(c *gin.Context) {
	var req types.BotRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	space, err := rh.botSpaceDB.GetByJoinCode(c, req.JoinCode)
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "invalid join code"})
			return
		}
		rh.log.WithError(err).Error("failed to lookup join code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	isManager := req.JoinCode == space.ManagerJoinCode
	caps := req.Capabilities

	now := time.Now()
	bot := types.Bot{
		ID:           uuid.New().String(),
		BotSpaceID:   space.ID,
		Name:         req.Name,
		Capabilities: &caps,
		IsManager:    isManager,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = rh.botDB.Insert(c, bot)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	if isManager {
		if err := rh.botSpaceDB.SetManagerBotID(c, space.ID, bot.ID); err != nil {
			rh.log.WithError(err).Error("failed to set manager bot id")
		}
	}

	token, err := rh.generateToken(&types.Claims{
		IsBot:      true,
		BotSpaceID: space.ID,
		BotID:      bot.ID,
		IsManager:  isManager,
	}, nil)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate bot token")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	c.JSON(http.StatusCreated, types.BotRegistrationResponse{
		Token: token,
		Bot:   bot,
		BotSpace: types.BotSpaceBasic{
			ID:   space.ID,
			Name: space.Name,
		},
	})
}
