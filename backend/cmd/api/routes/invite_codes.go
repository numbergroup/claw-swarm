package routes

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/server"
	ngerrors "github.com/numbergroup/errors"
)

func (rh *RouteHandler) CreateInviteCode(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	code, err := rh.generateCode(16)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate invite code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite code"})
		return
	}

	inviteCode := types.InviteCode{
		ID:         uuid.New().String(),
		BotSpaceID: botSpaceID,
		Code:       code,
		CreatedAt:  time.Now(),
	}

	_, err = rh.inviteCodeDB.Insert(c, inviteCode)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert invite code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite code"})
		return
	}

	c.JSON(http.StatusCreated, inviteCode)
}

func (rh *RouteHandler) ListInviteCodes(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	codes, err := rh.inviteCodeDB.ListByBotSpaceID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list invite codes")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list invite codes"})
		return
	}

	c.JSON(http.StatusOK, codes)
}

func (rh *RouteHandler) RevokeInviteCode(c *gin.Context) {
	_, _, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	codeID, err := server.GetUUIDParam(c, "codeId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid codeId"})
		return
	}

	if err := rh.inviteCodeDB.Delete(c, codeID.String()); err != nil {
		rh.log.WithError(err).Error("failed to revoke invite code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke invite code"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (rh *RouteHandler) JoinBotSpace(c *gin.Context) {
	claims := rh.getClaims(c)
	if claims == nil {
		return
	}
	if claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bots cannot join via invite codes"})
		return
	}

	var req types.JoinBotSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inviteCode, err := rh.inviteCodeDB.GetByCode(c, req.InviteCode)
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid invite code"})
			return
		}
		rh.log.WithError(err).Error("failed to lookup invite code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to join"})
		return
	}

	if inviteCode.ExpiresAt != nil && inviteCode.ExpiresAt.Before(time.Now()) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invite code has expired"})
		return
	}

	isMember, err := rh.spaceMemberDB.IsMember(c, inviteCode.BotSpaceID, claims.UserID)
	if err != nil {
		rh.log.WithError(err).Error("failed to check membership")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to join"})
		return
	}
	if isMember {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "already a member"})
		return
	}

	member := types.SpaceMember{
		ID:         uuid.New().String(),
		BotSpaceID: inviteCode.BotSpaceID,
		UserID:     claims.UserID,
		Role:       "member",
		JoinedAt:   time.Now(),
	}

	if _, err := rh.spaceMemberDB.Insert(c, member); err != nil {
		rh.log.WithError(err).Error("failed to insert member")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to join"})
		return
	}

	c.JSON(http.StatusOK, member)
}
