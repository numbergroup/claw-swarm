package routes

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) CreateBotSkill(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can create skills"})
		return
	}

	var req types.CreateBotSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot, err := rh.botDB.GetByID(c, claims.BotID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create skill"})
		return
	}

	now := time.Now()
	skill := types.BotSkill{
		ID:          uuid.New().String(),
		BotSpaceID:  botSpaceID,
		BotID:       claims.BotID,
		BotName:     bot.Name,
		Name:        req.Name,
		Description: req.Description,
		Tags:        pq.StringArray(req.Tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := rh.botSkillDB.Insert(c, skill)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "skill with this name already exists"})
			return
		}
		rh.log.WithError(err).Error("failed to insert bot skill")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create skill"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (rh *RouteHandler) ListBotSkills(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	skills, err := rh.botSkillDB.ListByBotSpaceID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list bot skills")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list skills"})
		return
	}

	c.JSON(http.StatusOK, skills)
}

func (rh *RouteHandler) UpdateBotSkill(c *gin.Context) {
	claims, _, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can update skills"})
		return
	}

	skillID, err := server.GetUUIDParam(c, "skillId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid skillId"})
		return
	}

	existing, err := rh.botSkillDB.GetByID(c, skillID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot skill")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update skill"})
		return
	}

	if existing.BotID != claims.BotID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you can only update your own skills"})
		return
	}

	var req types.UpdateBotSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Tags != nil {
		existing.Tags = pq.StringArray(req.Tags)
	}

	bot, err := rh.botDB.GetByID(c, claims.BotID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update skill"})
		return
	}
	existing.BotName = bot.Name
	existing.UpdatedAt = time.Now()

	result, err := rh.botSkillDB.Update(c, existing)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "skill with this name already exists"})
			return
		}
		rh.log.WithError(err).Error("failed to update bot skill")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update skill"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) DeleteBotSkill(c *gin.Context) {
	claims, _, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can delete skills"})
		return
	}

	skillID, err := server.GetUUIDParam(c, "skillId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid skillId"})
		return
	}

	existing, err := rh.botSkillDB.GetByID(c, skillID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot skill")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to delete skill"})
		return
	}

	if existing.BotID != claims.BotID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you can only delete your own skills"})
		return
	}

	if err := rh.botSkillDB.Delete(c, skillID.String()); err != nil {
		rh.log.WithError(err).Error("failed to delete bot skill")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to delete skill"})
		return
	}

	c.Status(http.StatusNoContent)
}
