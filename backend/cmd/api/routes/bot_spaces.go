package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
)

func (rh *RouteHandler) CreateBotSpace(c *gin.Context) {
	claims := rh.getClaims(c)
	if claims == nil {
		return
	}
	if claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bots cannot create bot spaces"})
		return
	}

	var req types.CreateBotSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	joinCode, err := rh.generateCode(16)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate join code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create bot space"})
		return
	}

	managerJoinCode, err := rh.generateCode(16)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate manager join code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create bot space"})
		return
	}

	now := time.Now()
	space := types.BotSpace{
		ID:              uuid.New().String(),
		OwnerID:         claims.UserID,
		Name:            req.Name,
		Description:     req.Description,
		JoinCode:        joinCode,
		ManagerJoinCode: managerJoinCode,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = rh.botSpaceDB.Insert(c, space)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert bot space")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create bot space"})
		return
	}

	member := types.SpaceMember{
		ID:         uuid.New().String(),
		BotSpaceID: space.ID,
		UserID:     claims.UserID,
		Role:       "owner",
		JoinedAt:   now,
	}
	if _, err := rh.spaceMemberDB.Insert(c, member); err != nil {
		rh.log.WithError(err).Error("failed to insert owner as member")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create bot space"})
		return
	}

	c.JSON(http.StatusCreated, space)
}

func (rh *RouteHandler) ListBotSpaces(c *gin.Context) {
	claims := rh.getClaims(c)
	if claims == nil {
		return
	}
	if claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bots cannot list bot spaces"})
		return
	}

	spaces, err := rh.botSpaceDB.ListByUserID(c, claims.UserID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list bot spaces")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list bot spaces"})
		return
	}

	c.JSON(http.StatusOK, spaces)
}

func (rh *RouteHandler) GetBotSpace(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	space, err := rh.botSpaceDB.GetByID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get bot space")
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot space not found"})
		return
	}

	if claims.IsBot {
		space.JoinCode = ""
		space.ManagerJoinCode = ""
	}

	c.JSON(http.StatusOK, space)
}

func (rh *RouteHandler) UpdateBotSpace(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	var req types.UpdateBotSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := rh.botSpaceDB.GetByID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get bot space")
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot space not found"})
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	desc := existing.Description
	if req.Description != nil {
		desc = req.Description
	}

	updated, err := rh.botSpaceDB.Update(c, botSpaceID, name, desc)
	if err != nil {
		rh.log.WithError(err).Error("failed to update bot space")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update bot space"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (rh *RouteHandler) DeleteBotSpace(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	if err := rh.botSpaceDB.Delete(c, botSpaceID); err != nil {
		rh.log.WithError(err).Error("failed to delete bot space")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to delete bot space"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (rh *RouteHandler) RegenerateJoinCodes(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	joinCode, err := rh.generateCode(16)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate join code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to regenerate join codes"})
		return
	}

	managerJoinCode, err := rh.generateCode(16)
	if err != nil {
		rh.log.WithError(err).Error("failed to generate manager join code")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to regenerate join codes"})
		return
	}

	updated, err := rh.botSpaceDB.UpdateJoinCodes(c, botSpaceID, joinCode, managerJoinCode)
	if err != nil {
		rh.log.WithError(err).Error("failed to update join codes")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to regenerate join codes"})
		return
	}

	c.JSON(http.StatusOK, updated)
}
