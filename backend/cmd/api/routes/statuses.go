package routes

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) ListStatuses(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	statuses, err := rh.botStatusDB.ListByBotSpaceID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list statuses")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list statuses"})
		return
	}

	c.JSON(http.StatusOK, statuses)
}

func (rh *RouteHandler) GetBotStatus(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	status, err := rh.botStatusDB.GetByBotSpaceIDAndBotID(c, botSpaceID, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot status not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot status")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get bot status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (rh *RouteHandler) UpdateBotStatus(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	var req types.UpdateBotStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot, err := rh.botDB.GetByID(c, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	now := time.Now()
	status := types.BotStatus{
		ID:             uuid.New().String(),
		BotSpaceID:     botSpaceID,
		BotID:          botID.String(),
		BotName:        bot.Name,
		Status:         req.Status,
		UpdatedByBotID: claims.BotID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	result, err := rh.botStatusDB.Upsert(c, status)
	if err != nil {
		rh.log.WithError(err).Error("failed to upsert bot status")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) BulkUpdateStatuses(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	var req types.BulkUpdateBotStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	statuses := make([]types.BotStatus, 0, len(req.Statuses))
	for _, item := range req.Statuses {
		bot, err := rh.botDB.GetByID(c, item.BotID)
		if err != nil {
			if ngerrors.Cause(err) == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found: " + item.BotID})
				return
			}
			rh.log.WithError(err).Error("failed to get bot")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update statuses"})
			return
		}

		statuses = append(statuses, types.BotStatus{
			ID:             uuid.New().String(),
			BotSpaceID:     botSpaceID,
			BotID:          item.BotID,
			BotName:        bot.Name,
			Status:         item.Status,
			UpdatedByBotID: claims.BotID,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	results, err := rh.botStatusDB.BulkUpsert(c, statuses)
	if err != nil {
		rh.log.WithError(err).Error("failed to bulk upsert statuses")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update statuses"})
		return
	}

	c.JSON(http.StatusOK, results)
}
