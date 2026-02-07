package routes

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
)

const managerReminder = "\n\n---\nReminder: If any bot statuses have changed based on the above summary, please submit status updates using the status update endpoint."

func (rh *RouteHandler) GetSummary(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	summary, err := rh.summaryDB.GetByBotSpaceID(c, botSpaceID)
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "no summary found"})
			return
		}
		rh.log.WithError(err).Error("failed to get summary")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get summary"})
		return
	}

	if claims.IsBot && claims.IsManager {
		summary.Content += managerReminder
	}

	c.JSON(http.StatusOK, summary)
}

func (rh *RouteHandler) UpdateSummary(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	var req types.UpdateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	summary := types.Summary{
		ID:             uuid.New().String(),
		BotSpaceID:     botSpaceID,
		Content:        req.Content,
		CreatedByBotID: claims.BotID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	result, err := rh.summaryDB.Upsert(c, summary)
	if err != nil {
		rh.log.WithError(err).Error("failed to upsert summary")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to update summary"})
		return
	}

	c.JSON(http.StatusOK, result)
}
